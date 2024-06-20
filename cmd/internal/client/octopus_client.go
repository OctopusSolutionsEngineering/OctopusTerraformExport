package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/avast/retry-go/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type OctopusClient interface {
	GetSpaceBaseUrl() (string, error)
	GetSpace(resources *octopus2.Space) error
	GetSpaces() ([]octopus2.Space, error)
	EnsureSpaceDeleted(spaceId string) (deleted bool, funcErr error)
	GetResource(resourceType string, resources any) (exists bool, funcErr error)
	GetResourceById(resourceType string, id string, resources any) (exists bool, funcErr error)
	GetResourceNameById(resourceType string, id string) (name string, funcErr error)
	GetResourceNamesByIds(resourceType string, id []string) (name []string, funcErr error)
	GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error)
}

type OctopusApiClient struct {
	Url    string
	ApiKey string
	Space  string
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

func (o *OctopusApiClient) lookupSpaceAsId() (bool, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return false, errors.New("space can not be empty")
	}

	requestURL := fmt.Sprintf("%s/api/Spaces/%s", o.Url, o.Space)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return false, err
	}

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	}

	return res.StatusCode != 404, nil
}

func (o *OctopusApiClient) lookupSpaceAsName() (spaceName string, funcErr error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("space can not be empty")
	}

	requestURL := fmt.Sprintf("%s/api/Spaces?take=1000&partialName=%s", o.Url, url.QueryEscape(o.Space))

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return "", err
	}

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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

	collection := octopus2.GeneralCollection[octopus2.Space]{}
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

	if o.spaceId != "" {
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, o.spaceId), nil
	}

	spaceId, err := o.lookupSpaceAsName()
	if err == nil {
		o.spaceId = spaceId
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, spaceId), nil
	}

	spaceIdValid, err := o.lookupSpaceAsId()
	if spaceIdValid && err == nil {
		o.spaceId = o.Space
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, o.Space), nil
	}

	return "", errors.New("getSpaceUrl did not find space with name or id '" + o.Space + "'")
}

func (o *OctopusApiClient) GetSpaceBaseUrl() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("GetSpaceBaseUrl - space can not be empty")
	}

	// Sometimes looking up a space that was just created failed, so add a retry
	return retry.DoWithData(func() (string, error) {
		o.mu.Lock()
		defer o.mu.Unlock()

		if o.spaceId != "" {
			return fmt.Sprintf("%s/api/%s", o.Url, o.spaceId), nil
		}

		spaceId, err := o.lookupSpaceAsName()
		if err == nil {
			o.spaceId = spaceId
			return fmt.Sprintf("%s/api/%s", o.Url, spaceId), nil
		}

		spaceIdValid, err := o.lookupSpaceAsId()
		if spaceIdValid && err == nil {
			o.spaceId = o.Space
			return fmt.Sprintf("%s/api/%s", o.Url, o.Space), nil
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

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
	}

	return req, nil
}

func (o *OctopusApiClient) getRequest(resourceType string, id string) (*http.Request, error) {
	spaceUrl, err := o.GetSpaceBaseUrl()

	if err != nil {
		return nil, err
	}

	requestURL := spaceUrl + "/" + resourceType + "/" + id

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return nil, err
	}

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
	}

	return req, nil
}

func (o *OctopusApiClient) GetSpace(resources *octopus2.Space) (funcErr error) {
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

func (o *OctopusApiClient) GetSpaces() (spaces []octopus2.Space, funcErr error) {
	requestURL := fmt.Sprintf("%s/api/Spaces", o.Url)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return nil, err
	}

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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

	collection := octopus2.GeneralCollection[octopus2.Space]{}
	err = json.NewDecoder(res.Body).Decode(&collection)

	if err != nil {
		return nil, err
	}

	return collection.Items, nil
}

func (o *OctopusApiClient) EnsureSpaceDeleted(spaceId string) (deleted bool, funcErr error) {
	requestURL := fmt.Sprintf("%s/api/Spaces/%s", o.Url, spaceId)

	// Get the details of the space
	space, err := func() (*octopus2.Space, error) {
		getReq, err := http.NewRequest(http.MethodGet, requestURL, nil)

		if err != nil {
			return nil, err
		}

		if o.ApiKey != "" {
			getReq.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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

		space := octopus2.Space{}
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

		if o.ApiKey != "" {
			putReq.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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

		if o.ApiKey != "" {
			req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
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
		errorResponse := octopus2.ErrorResponse{}
		err = json.Unmarshal(body, &errorResponse)

		if err == nil {
			// treat a missing variables.ocl file as a missing resource
			if strings.Index(errorResponse.ErrorMessage, "does not exist in the Git repository") != -1 && strings.Index(errorResponse.ErrorMessage, "variables.ocl") != -1 {
				return false, nil
			}
		}

		return false, errors.New("did not find the requested resource: " + resourceType + "\n" + fmt.Sprint(res.StatusCode) + "\n" + string(body[:]))
	}

	err = json.Unmarshal(body, resources)

	if err != nil {
		zap.L().Error(string(body))
		return false, err
	}

	return true, nil
}

func (o *OctopusApiClient) GetResourceById(resourceType string, id string, resources any) (exists bool, funcErr error) {
	cacheHit := o.readCache(resourceType, id)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType + " " + id)

		err := o.unmarshal(resources, cacheHit)

		if err != nil {
			return false, err
		}

		return true, nil
	}

	zap.L().Debug("Getting " + resourceType + " " + id)

	req, err := o.getRequest(resourceType, id)

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
		return false, err
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
	nameId := octopus2.NameId{}
	cacheHit := o.readCache(resourceType, id)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType + " " + id)

		err := o.unmarshal(&nameId, cacheHit)

		if err != nil {
			return "", err
		}

		return nameId.Name, nil
	}

	zap.L().Debug("Getting " + resourceType + " " + id)

	req, err := o.getRequest(resourceType, id)

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
		return "", err
	}

	return nameId.Name, nil
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
		return err
	}

	return nil
}

func (o *OctopusApiClient) GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error) {
	queryParamsId := strings.Join(lo.Map(queryParams, func(item []string, index int) string {
		return item[0] + "=" + item[1]
	}), ",")

	cacheId := resourceType + "[" + queryParamsId + "]"

	cacheHit := o.readCollectionCache(cacheId)
	if cacheHit != nil {
		zap.L().Debug("Cache hit on " + resourceType)

		err := json.Unmarshal(cacheHit, resources)

		if err != nil {
			return err
		}

		return nil
	}

	zap.L().Debug("Getting collection " + resourceType)

	req, err := o.getCollectionRequest(resourceType, queryParams...)

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

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return err
	}

	o.cacheCollectionResult(resourceType, cacheId, body)

	return o.unmarshal(resources, body)
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
	// Only worker pools and tag sets are looked up by other resources. Worker pools may be looked
	// up to find the default worker pool. Tag sets are looked up to add terraform dependencies.
	if resourceType != "WorkerPools" && resourceType != "TagSets" && resourceType != "Environments" {
		return
	}

	o.collectionCacheMu.Lock()
	defer o.collectionCacheMu.Unlock()

	if o.collectionCache == nil {
		o.collectionCache = map[string][]byte{}
	}

	o.collectionCache[cacheId] = body
}
