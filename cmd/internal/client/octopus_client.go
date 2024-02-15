package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/avast/retry-go/v4"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OctopusClient interface {
	GetSpaceBaseUrl() (string, error)
	GetSpace(resources *octopus2.Space) error
	GetSpaces() ([]octopus2.Space, error)
	EnsureSpaceDeleted(spaceId string) (funcErr error)
	GetResource(resourceType string, resources any) (exists bool, funcErr error)
	GetResourceById(resourceType string, id string, resources any) (exists bool, funcErr error)
	GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error)
}

type OctopusApiClient struct {
	Url    string
	ApiKey string
	Space  string
}

func (o OctopusApiClient) lookupSpaceAsId() (bool, error) {
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

func (o OctopusApiClient) lookupSpaceAsName() (spaceName string, funcErr error) {
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

func (o OctopusApiClient) getSpaceUrl() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("getSpaceUrl - space can not be empty")
	}

	spaceId, err := o.lookupSpaceAsName()
	if err == nil {
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, spaceId), nil
	}

	spaceIdValid, err := o.lookupSpaceAsId()
	if spaceIdValid && err == nil {
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, o.Space), nil
	}

	return "", errors.New("getSpaceUrl did not find space with name or id '" + o.Space + "'")
}

func (o OctopusApiClient) GetSpaceBaseUrl() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("GetSpaceBaseUrl - space can not be empty")
	}

	// Sometimes looking up a space that was just created failed, so add a retry
	return retry.DoWithData(func() (string, error) {
		spaceId, err := o.lookupSpaceAsName()
		if err == nil {
			return fmt.Sprintf("%s/api/%s", o.Url, spaceId), nil
		}

		spaceIdValid, err := o.lookupSpaceAsId()
		if spaceIdValid && err == nil {
			return fmt.Sprintf("%s/api/%s", o.Url, o.Space), nil
		}

		return "", errors.New("GetSpaceBaseUrl did not find space with name or id '" + o.Space + "'")
	}, retry.Attempts(3), retry.Delay(1*time.Second))
}

func (o OctopusApiClient) getSpaceRequest() (*http.Request, error) {
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

func (o OctopusApiClient) getRequest(resourceType string, id string) (*http.Request, error) {
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

func (o OctopusApiClient) getCollectionRequest(resourceType string, queryParams ...[]string) (*http.Request, error) {
	spaceUrl, err := o.GetSpaceBaseUrl()

	if err != nil {
		return nil, err
	}

	requestURL := spaceUrl + "/" + resourceType + "?take=10000"

	for _, q := range queryParams {

		if len(q) == 1 {
			requestURL += "&" + url.QueryEscape(q[0])
		}

		if len(q) == 2 {
			requestURL += "&" + url.QueryEscape(q[0]) + "=" + url.QueryEscape(q[1])
		}
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return nil, err
	}

	if o.ApiKey != "" {
		req.Header.Set("X-Octopus-ApiKey", o.ApiKey)
	}

	return req, nil
}

func (o OctopusApiClient) GetSpace(resources *octopus2.Space) (funcErr error) {
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

func (o OctopusApiClient) GetSpaces() (spaces []octopus2.Space, funcErr error) {
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

func (o OctopusApiClient) EnsureSpaceDeleted(spaceId string) (funcErr error) {
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
		return err
	}

	if space == nil {
		return nil
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
		return err
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

	return err
}

func (o OctopusApiClient) bodyToString(body io.Reader) (string, error) {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (o OctopusApiClient) GetResource(resourceType string, resources any) (exists bool, funcErr error) {
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
		return false, errors.New("did not find the requested resource: " + resourceType + "\n" + string(body[:]))
	}

	err = json.Unmarshal(body, resources)

	if err != nil {
		zap.L().Error(string(body))
		return false, err
	}

	return true, nil
}

func (o OctopusApiClient) GetResourceById(resourceType string, id string, resources any) (exists bool, funcErr error) {
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

	err = json.Unmarshal(body, resources)

	if err != nil {
		zap.L().Error(string(body))
		return false, err
	}

	return true, nil
}

func (o OctopusApiClient) GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error) {
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

	return json.NewDecoder(res.Body).Decode(resources)
}
