package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/avast/retry-go/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type OctopusClient interface {
	GetSpaceBaseUrl() (string, error)
	GetSpace(resources *octopus.Space) error
	GetSpaces() ([]octopus.Space, error)
	EnsureSpaceDeleted(spaceId string) (deleted bool, funcErr error)
	GetResource(resourceType string, resources any) (exists bool, funcErr error)
	GetResourceById(resourceType string, id string, resources any) (funcErr error)
	GetResourceByName(resourceType string, name string, resources any) (exists bool, funcErr error)
	GetSpaceResourceById(resourceType string, id string, resources any) (exists bool, funcErr error)
	GetGlobalResourceById(resourceType string, id string, resources any) (exists bool, funcErr error)
	GetResourceNameById(resourceType string, id string) (name string, funcErr error)
	GetResourceNamesByIds(resourceType string, id []string) (name []string, funcErr error)
	GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error)
	GetAllGlobalResources(resourceType string, resources any, queryParams ...[]string) (funcErr error)
}

type OctopusApiClient struct {
	Url         string
	ApiKey      string
	AccessToken string
	Space       string
	Version     string
	// a flag to indicate if the client must use the redirector
	// https://github.com/OctopusSolutionsEngineering/AzureFunctionRouter
	UseRedirector bool
	// RedirectorHost is the hostname of the redirector to use
	RedirectorHost string
	// This key must be known by the service that is using the redirector
	RedirectorServiceApiKey string
	// This key is passed by the caller
	RedirecrtorApiKey string
	// These rules are passed by the called
	RedirectorRedirections string
	// spaceId is what Space resolves to after a lookup. We cache the result to save on future lookups.
	spaceId string
	// mu is the mutex to lock the update of the SpaceId parameter
	mu sync.Mutex
	// cache is a map of resource types to a map of ids with the resource as a string
	cache   map[string]map[string][]byte
	cacheMu sync.Mutex
	// collectionCache is a map of resource types to their collections
	collectionCache   map[string][]byte
	collectionCacheMu sync.Mutex
}

func (o *OctopusApiClient) buildUserAgent() string {
	if o.Version == "" {
		return "octoterra"
	}

	return "octoterra/" + o.Version + " (" + runtime.GOOS + " " + runtime.GOARCH + ")"
}

func (o *OctopusApiClient) buildUrl() (string, error) {
	if o.UseRedirector {
		if o.RedirectorHost == "" {
			return "", errors.New("RedirectorHost must be set when UseRedirector is true")
		}

		if o.RedirectorServiceApiKey == "" {
			return "", errors.New("RedirectorServiceApiKey must be set when UseRedirector is true")
		}

		return "https://" + o.RedirectorHost, nil
	}

	return o.Url, nil
}

func (o *OctopusApiClient) lookupSpaceAsId() (bool, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return false, errors.New("space can not be empty")
	}

	baseUrl, err := o.buildUrl()

	if err != nil {
		return false, err
	}

	requestURL := fmt.Sprintf("%s/api/Spaces/%s", baseUrl, o.Space)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return false, err
	}

	if err := o.setHeaders(req); err != nil {
		return false, err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	}

	return res.StatusCode != 404, nil
}

func (o *OctopusApiClient) setHeaders(req *http.Request) error {
	if strings.TrimSpace(o.ApiKey) != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
	} else if strings.TrimSpace(o.AccessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+o.AccessToken)
	}

	// See https://github.com/OctopusSolutionsEngineering/AzureFunctionRouter
	if o.UseRedirector {

		parsedUrl, err := url.Parse(o.Url)

		if err != nil {
			return err
		}

		req.Header.Set("X_REDIRECTION_UPSTREAM_HOST", parsedUrl.Hostname())
		req.Header.Set("X_REDIRECTION_REDIRECTIONS", o.RedirectorRedirections)
		req.Header.Set("X_REDIRECTION_API_KEY", o.RedirecrtorApiKey)
		req.Header.Set("X_REDIRECTION_SERVICE_API_KEY", o.RedirectorServiceApiKey)
	}

	req.Header.Set("User-Agent", o.buildUserAgent())

	return nil
}

func (o *OctopusApiClient) lookupSpaceAsName() (spaceName string, funcErr error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("space can not be empty")
	}

	baseUrl, err := o.buildUrl()

	if err != nil {
		return "", err
	}

	requestURL := fmt.Sprintf("%s/api/Spaces?take=1000&partialName=%s", baseUrl, url.QueryEscape(o.Space))

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return "", err
	}

	if err := o.setHeaders(req); err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		zap.L().Error(err.Error())
		return "", err
	}

	if res.StatusCode != 200 {
		return "", nil
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	collection := octopus.GeneralCollection[octopus.Space]{}
	err = json.NewDecoder(res.Body).Decode(&collection)

	if err != nil {
		return "", err
	}

	for _, space := range collection.Items {
		if space.Name == o.Space {
			return space.Id, nil
		}
	}

	return "", errors.New("did not find space with name " + o.Space)
}

func (o *OctopusApiClient) getSpaceUrl() (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("getSpaceUrl - space can not be empty")
	}

	baseUrl, err := o.buildUrl()

	if err != nil {
		return "", err
	}

	if o.spaceId != "" {
		return fmt.Sprintf("%s/api/Spaces/%s", baseUrl, o.spaceId), nil
	}

	spaceId, err := o.lookupSpaceAsName()
	if err == nil {
		o.spaceId = spaceId
		return fmt.Sprintf("%s/api/Spaces/%s", baseUrl, spaceId), nil
	}

	spaceIdValid, err := o.lookupSpaceAsId()
	if spaceIdValid && err == nil {
		o.spaceId = o.Space
		return fmt.Sprintf("%s/api/Spaces/%s", baseUrl, o.Space), nil
	}

	return "", errors.New("getSpaceUrl did not find space with name or id '" + o.Space + "'")
}

func (o *OctopusApiClient) GetBaseUrl() (string, error) {
	baseUrl, err := o.buildUrl()

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/api", baseUrl), nil
}

func (o *OctopusApiClient) GetSpaceBaseUrl() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("GetSpaceBaseUrl - space can not be empty")
	}

	// Sometimes looking up a space that was just created failed, so add a retry
	return retry.DoWithData(func() (string, error) {
		o.mu.Lock()
		defer o.mu.Unlock()

		baseUrl, err := o.buildUrl()

		if err != nil {
			return "", err
		}

		if o.spaceId != "" {
			return fmt.Sprintf("%s/api/%s", baseUrl, o.spaceId), nil
		}

		spaceId, err := o.lookupSpaceAsName()
		if err == nil {
			o.spaceId = spaceId
			return fmt.Sprintf("%s/api/%s", baseUrl, spaceId), nil
		}

		spaceIdValid, err := o.lookupSpaceAsId()
		if spaceIdValid && err == nil {
			o.spaceId = o.Space
			return fmt.Sprintf("%s/api/%s", baseUrl, o.Space), nil
		}

		return "", errors.New("GetSpaceBaseUrl did not find space with name or id '" + o.Space + "'")
	}, retry.Attempts(3), retry.Delay(1*time.Second))
}

func (o *OctopusApiClient) getSpaceRequest() (*http.Request, error) {
	spaceUrl, err := o.getSpaceUrl()

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, spaceUrl, nil)

	if err != nil {
		return nil, err
	}

	if err := o.setHeaders(req); err != nil {
		return nil, err
	}

	return req, nil
}

func (o *OctopusApiClient) getRequest(resourceType string, id string, global bool) (*http.Request, error) {
	spaceUrl, err := func() (string, error) {
		if global {
			return o.GetBaseUrl()
		}

		return o.GetSpaceBaseUrl()
	}()

	if err != nil {
		return nil, err
	}

	requestURL := spaceUrl + "/" + resourceType + "/" + id

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return nil, err
	}

	if err := o.setHeaders(req); err != nil {
		return nil, err
	}

	return req, nil
}

func (o *OctopusApiClient) getCollectionRequest(resourceType string, queryParams ...[]string) (*http.Request, error) {
	spaceUrl, err := o.GetSpaceBaseUrl()

	if err != nil {
		return nil, err
	}

	requestURL, err := url.Parse(spaceUrl + "/" + resourceType)

	if err != nil {
		panic(err)
	}

	params := url.Values{}
	for _, q := range queryParams {
		if len(q) == 1 {
			params.Add(q[0], "")
		}

		if len(q) == 2 {
			params.Add(q[0], q[1])
		}
	}

	// Add default take query param if it was not specified
	if _, ok := params["take"]; !ok {
		params.Add("take", "10000")
	}

	requestURL.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, requestURL.String(), nil)

	if err != nil {
		return nil, err
	}

	if err := o.setHeaders(req); err != nil {
		return nil, err
	}

	return req, nil
}

func (o *OctopusApiClient) getGlobalCollectionRequest(resourceType string, queryParams ...[]string) (*http.Request, error) {
	spaceUrl, err := o.GetBaseUrl()

	if err != nil {
		return nil, err
	}

	requestURL, err := url.Parse(spaceUrl + "/" + resourceType)

	if err != nil {
		panic(err)
	}

	params := url.Values{}
	for _, q := range queryParams {
		if len(q) == 1 {
			params.Add(q[0], "")
		}

		if len(q) == 2 {
			params.Add(q[0], q[1])
		}
	}

	// Add default take query param if it was not specified
	if _, ok := params["take"]; !ok {
		params.Add("take", "10000")
	}

	requestURL.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, requestURL.String(), nil)

	if err != nil {
		return nil, err
	}

	if err := o.setHeaders(req); err != nil {
		return nil, err
	}

	return req, nil
}

func (o *OctopusApiClient) GetSpace(resources *octopus.Space) (funcErr error) {
	req, err := o.getSpaceRequest()

	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return nil
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	return json.NewDecoder(res.Body).Decode(resources)
}

func (o *OctopusApiClient) GetSpaces() (spaces []octopus.Space, funcErr error) {
	baseUrl, err := o.buildUrl()

	if err != nil {
		return nil, err
	}

	requestURL := fmt.Sprintf("%s/api/Spaces", baseUrl)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return nil, err
	}

	if err := o.setHeaders(req); err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		zap.L().Error(err.Error())
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("Status code was " + fmt.Sprint(res.StatusCode) + ".")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	collection := octopus.GeneralCollection[octopus.Space]{}
	err = json.NewDecoder(res.Body).Decode(&collection)

	if err != nil {
		return nil, err
	}

	return collection.Items, nil
}

func (o *OctopusApiClient) EnsureSpaceDeleted(spaceId string) (deleted bool, funcErr error) {
	baseUrl, err := o.buildUrl()

	if err != nil {
		return false, err
	}

	requestURL := fmt.Sprintf("%s/api/Spaces/%s", baseUrl, spaceId)

	// Get the details of the space
	space, err := func() (*octopus.Space, error) {
		getReq, err := http.NewRequest(http.MethodGet, requestURL, nil)

		if err != nil {
			return nil, err
		}

		if err := o.setHeaders(getReq); err != nil {
			return nil, err
		}

		getRes, err := http.DefaultClient.Do(getReq)

		if err != nil {
			return nil, err
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				funcErr = errors.Join(funcErr, err)
			}
		}(getRes.Body)

		// If the space doesn't exist, there is nothing left to do
		if getRes.StatusCode == 404 {
			return nil, nil
		}

		if getRes.StatusCode != 200 {
			body, _ := o.bodyToString(getRes.Body)
			return nil, errors.New("Status code was " + fmt.Sprint(getRes.StatusCode) + " with body " + body)
		}

		space := octopus.Space{}
		err = json.NewDecoder(getRes.Body).Decode(&space)

		if err != nil {
			return nil, err
		}

		return &space, nil
	}()

	if err != nil {
		return false, err
	}

	if space == nil {
		return false, nil
	}

	// disable task processing
	err = func() error {
		space.TaskQueueStopped = true
		spaceJson, err := json.Marshal(&space)

		if err != nil {
			return err
		}

		putReq, err := http.NewRequest(http.MethodPut, requestURL, bytes.NewReader(spaceJson))

		if err != nil {
			return err
		}

		if err := o.setHeaders(putReq); err != nil {
			return err
		}

		putRes, err := http.DefaultClient.Do(putReq)

		if err != nil {
			return err
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				funcErr = errors.Join(funcErr, err)
			}
		}(putRes.Body)

		if putRes.StatusCode != 200 {
			body, _ := o.bodyToString(putRes.Body)
			return errors.New("Status code was " + fmt.Sprint(putRes.StatusCode) + " with body " + body)
		}

		return nil
	}()

	if err != nil {
		return false, err
	}

	// Delete the space
	err = func() error {
		req, err := http.NewRequest(http.MethodDelete, requestURL, nil)

		if err != nil {
			return err
		}

		if err := o.setHeaders(req); err != nil {
			return err
		}

		res, err := http.DefaultClient.Do(req)

		if err != nil {
			return err
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				funcErr = errors.Join(funcErr, err)
			}
		}(res.Body)

		if res.StatusCode != 200 {
			body, _ := o.bodyToString(res.Body)
			return errors.New("Status code was " + fmt.Sprint(res.StatusCode) + " with body " + body)
		}

		return nil
	}()

	return true, err
}

func (o *OctopusApiClient) bodyToString(body io.Reader) (string, error) {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (o *OctopusApiClient) GetResource(resourceType string, resources any) (exists bool, funcErr error) {
	zap.L().Debug("Getting " + resourceType)

	spaceUrl, err := o.GetSpaceBaseUrl()

	if err != nil {
		return false, err
	}

	requestURL := spaceUrl + "/" + resourceType

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return false, err
	}

	if err := o.setHeaders(req); err != nil {
		return false, err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	if res.StatusCode == 404 {
		return false, nil
	}

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return false, err
	}

	if res.StatusCode != 200 {
		errorResponse := octopus.ErrorResponse{}
		err = json.Unmarshal(body, &errorResponse)

		if err == nil {

			if strings.Index(errorResponse.ErrorMessage, "does not exist in the Git repository") != -1 && strings.Index(errorResponse.ErrorMessage, "variables.ocl") != -1 {
				// treat a missing variables.ocl file as a missing resource
				return false, nil
			} else if strings.Index(errorResponse.ErrorMessage, "Invalid username or password") != -1 {
				// Treat invalid git credentials as a missing resource
				return false, nil
			} else if strings.Index(errorResponse.ErrorMessage, "Support for password authentication was removed") != -1 {
				// Ignore this error message:
				// Support for password authentication was removed on August 13, 2021.\nPlease see https://docs.github.com/get-started/getting-started-with-git/about-remote-repositories#cloning-with-https-urls for information on currently recommended modes of authentication.
				return false, nil
			}
		}

		return false, errors.New("did not find the requested resource: " + resourceType + "\n" + fmt.Sprint(res.StatusCode) + "\n" + string(body[:]))
	}

	err = json.Unmarshal(body, resources)

	if err != nil {
		zap.L().Error(string(body))
		return false, fmt.Errorf("error in OctopusApiClient.GetResource loading resource type %s into type %s from %s: %w",
			resourceType, reflect.TypeOf(resources).String(), body, err)
	}

	return true, nil
}

func (o *OctopusApiClient) GetSpaceResourceById(resourceType string, id string, resources any) (exists bool, funcErr error) {
	return o.getResourceById(resourceType, false, id, resources)
}

func (o *OctopusApiClient) GetResourceByName(resourceType string, name string, resource any) (exists bool, funcErr error) {
	collection := octopus.GeneralCollection[octopus.NameId]{}
	if err := o.GetAllResources(resourceType, &collection, []string{"partialName", name}, []string{"take", "10000"}); err != nil {
		return false, err
	}

	item := lo.Filter(collection.Items, func(item octopus.NameId, index int) bool {
		return item.Name == name
	})

	if len(item) == 1 {
		return true, o.GetResourceById(resourceType, item[0].Id, resource)
	}

	return false, nil
}

func (o *OctopusApiClient) GetGlobalResourceById(resourceType string, id string, resources any) (exists bool, funcErr error) {
	return o.getResourceById(resourceType, true, id, resources)
}

func (o *OctopusApiClient) getResourceById(resourceType string, global bool, id string, resources any) (exists bool, funcErr error) {
	cacheHit := o.readCache(resourceType, id)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType + " " + id)

		err := o.unmarshal(resources, cacheHit)

		if err != nil {
			return false, fmt.Errorf("error in OctopusApiClient.getResourceById loading resource type %s with id %s into type %s from %s: %w",
				resourceType, id, reflect.TypeOf(resources).String(), cacheHit, err)
		}

		return true, nil
	}

	zap.L().Debug("Getting " + resourceType + " " + id)

	req, err := o.getRequest(resourceType, id, global)

	if err != nil {
		return false, err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	}

	if res.StatusCode == 404 {
		return false, nil
	}

	if res.StatusCode != 200 {
		return false, errors.New("did not find the requested resource: " + resourceType + " " + id)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return false, err
	}

	o.cacheResult(resourceType, id, body)

	err = o.unmarshal(resources, body)

	if err != nil {
		return false, fmt.Errorf("error in OctopusApiClient.getResourceById loading resource type %s with id %s into type %s from %s: %w",
			resourceType, id, reflect.TypeOf(resources).String(), body, err)
	}

	return true, nil
}

func (o *OctopusApiClient) GetResourceNamesByIds(resourceType string, id []string) (names []string, funcErr error) {
	var mappingErrors error = nil
	resourceNames := lo.Map(id, func(item string, index int) string {
		name, err := o.GetResourceNameById(resourceType, item)
		if err != nil {
			mappingErrors = errors.Join(mappingErrors, err)
		}
		return name
	})

	return resourceNames, mappingErrors
}

func (o *OctopusApiClient) GetResourceNameById(resourceType string, id string) (name string, funcErr error) {
	nameId := octopus.NameId{}
	cacheHit := o.readCache(resourceType, id)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType + " " + id)

		err := o.unmarshal(&nameId, cacheHit)

		if err != nil {
			return "", fmt.Errorf("error in OctopusApiClient.GetResourceNameById loading resource type %s with id %s into type %s from %s: %w",
				resourceType, id, reflect.TypeOf(nameId).String(), cacheHit, err)
		}

		return nameId.Name, nil
	}

	zap.L().Debug("Getting " + resourceType + " " + id)

	req, err := o.getRequest(resourceType, id, false)

	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	if res.StatusCode == 404 {
		return "", nil
	}

	if res.StatusCode != 200 {
		return "", errors.New("did not find the requested resource: " + resourceType + " " + id)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	o.cacheResult(resourceType, id, body)

	err = o.unmarshal(&nameId, body)

	if err != nil {
		return "", fmt.Errorf("error in OctopusApiClient.GetResourceNameById loading resource type %s with id %s into type %s from %s: %w",
			resourceType, id, reflect.TypeOf(nameId).String(), body, err)
	}

	return nameId.Name, nil
}

func (o *OctopusApiClient) GetResourceById(resourceType string, id string, resources any) (funcErr error) {
	cacheHit := o.readCache(resourceType, id)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType + " " + id)

		err := o.unmarshal(&resources, cacheHit)

		if err != nil {
			return fmt.Errorf("error in OctopusApiClient.GetResourceById loading resource type %s with id %s into type %s from %s: %w",
				resourceType, id, reflect.TypeOf(resources).String(), cacheHit, err)
		}

		return nil
	}

	zap.L().Debug("Getting " + resourceType + " " + id)

	req, err := o.getRequest(resourceType, id, false)

	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	if res.StatusCode == 404 {
		return nil
	}

	if res.StatusCode != 200 {
		return errors.New("did not find the requested resource: " + resourceType + " " + id)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return err
	}

	o.cacheResult(resourceType, id, body)

	err = o.unmarshal(resources, body)

	if err != nil {
		return fmt.Errorf("error in OctopusApiClient.GetResourceNameById loading resource type %s with id %s into type %s from %s: %w",
			resourceType, id, reflect.TypeOf(resources).String(), body, err)
	}

	return nil
}

func (o *OctopusApiClient) readCache(resourceType string, id string) []byte {
	o.cacheMu.Lock()
	defer o.cacheMu.Unlock()

	if val, ok := o.cache[resourceType]; ok {
		if val, ok := val[id]; ok {
			return val
		}
	}

	return nil
}

func (o *OctopusApiClient) cacheResult(resourceType string, id string, body []byte) {
	// Only projects and tenants are resolved by other resources. Tenant variables lookup tenants
	// to see if they have been excluded. Many resources look up projects to see if they exist and
	// if they have been excluded. Caching these resources saves repeat calls to the API.
	if resourceType != "Environments" && resourceType != "Projects" && resourceType != "Tenants" && resourceType != "LibraryVariableSets" {
		return
	}

	o.cacheMu.Lock()
	defer o.cacheMu.Unlock()

	if o.cache == nil {
		o.cache = map[string]map[string][]byte{}
	}

	if _, ok := o.cache[resourceType]; !ok {
		o.cache[resourceType] = map[string][]byte{}
	}

	o.cache[resourceType][id] = body
}

func (o *OctopusApiClient) unmarshal(resources any, body []byte) error {
	err := json.Unmarshal(body, resources)

	if err != nil {
		zap.L().Error(string(body))
		return fmt.Errorf("error in OctopusApiClient.unmarshal loading resource into type %s from %s: %w",
			reflect.TypeOf(resources).String(), body, err)
	}

	return nil
}

func (o *OctopusApiClient) GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error) {
	req, err := o.getCollectionRequest(resourceType, queryParams...)

	if err != nil {
		return err
	}

	return o.getAllResources(req, resourceType, resources, queryParams...)
}

func (o *OctopusApiClient) GetAllGlobalResources(resourceType string, resources any, queryParams ...[]string) (funcErr error) {
	req, err := o.getGlobalCollectionRequest(resourceType, queryParams...)

	if err != nil {
		return err
	}

	return o.getAllResources(req, resourceType, resources, queryParams...)
}

func (o *OctopusApiClient) getAllResources(req *http.Request, resourceType string, resources any, queryParams ...[]string) (funcErr error) {
	queryParamsId := strings.Join(lo.Map(queryParams, func(item []string, index int) string {
		return item[0] + "=" + item[1]
	}), ",")

	cacheId := resourceType + "[" + queryParamsId + "]"

	cacheHit := o.readCollectionCache(cacheId)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType)

		err := json.Unmarshal(cacheHit, resources)

		if err != nil {
			return fmt.Errorf("error in OctopusApiClient.getAllResources loading resource type %s into type %s from %s: %w",
				resourceType, reflect.TypeOf(resources).String(), cacheHit, err)
		}

		return nil
	}

	zap.L().Debug("Getting collection " + resourceType)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return nil
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			funcErr = errors.Join(funcErr, err)
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return err
	}

	o.cacheCollectionResult(resourceType, cacheId, body)

	err = o.unmarshal(resources, body)

	if err != nil {
		return fmt.Errorf("error in OctopusApiClient.getAllResources loading resource type %s into type %s from %s: %w",
			resourceType, reflect.TypeOf(resources).String(), body, err)
	}

	return nil
}

func (o *OctopusApiClient) readCollectionCache(cacheId string) []byte {
	o.collectionCacheMu.Lock()
	defer o.collectionCacheMu.Unlock()

	if val, ok := o.collectionCache[cacheId]; ok {
		return val
	}

	return nil
}

func (o *OctopusApiClient) cacheCollectionResult(resourceType string, cacheId string, body []byte) {
	// Only some collections are looked up by other resources. Worker pools may be looked
	// up to find the default worker pool. Tag sets are looked up to add terraform dependencies.
	// Environments are exported by projects.
	if !lo.Contains([]string{"WorkerPools", "TagSets", "Environments"}, resourceType) {
		return
	}

	o.collectionCacheMu.Lock()
	defer o.collectionCacheMu.Unlock()

	if o.collectionCache == nil {
		o.collectionCache = map[string][]byte{}
	}

	o.collectionCache[cacheId] = body
}
