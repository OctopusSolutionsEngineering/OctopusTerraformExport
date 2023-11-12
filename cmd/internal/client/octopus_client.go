package client

import (
	"encoding/json"
	"errors"
	"fmt"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type OctopusClient struct {
	Url    string
	ApiKey string
	Space  string
}

func (o OctopusClient) lookupSpaceAsId() (bool, error) {
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

func (o OctopusClient) lookupSpaceAsName() (spaceName string, funcErr error) {
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

func (o OctopusClient) getSpaceUrl() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("space can not be empty")
	}

	spaceId, err := o.lookupSpaceAsName()
	if err == nil {
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, spaceId), nil
	}

	spaceIdValid, err := o.lookupSpaceAsId()
	if spaceIdValid && err == nil {
		return fmt.Sprintf("%s/api/Spaces/%s", o.Url, o.Space), nil
	}

	return "", errors.New("did not find space with name or id '" + o.Space + "'")
}

func (o OctopusClient) GetSpaceBaseUrl() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("space can not be empty")
	}

	spaceId, err := o.lookupSpaceAsName()
	if err == nil {
		return fmt.Sprintf("%s/api/%s", o.Url, spaceId), nil
	}

	spaceIdValid, err := o.lookupSpaceAsId()
	if spaceIdValid && err == nil {
		return fmt.Sprintf("%s/api/%s", o.Url, o.Space), nil
	}

	return "", errors.New("did not find space with name or id '" + o.Space + "'")
}

func (o OctopusClient) getSpaceRequest() (*http.Request, error) {
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

func (o OctopusClient) getRequest(resourceType string, id string) (*http.Request, error) {
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

func (o OctopusClient) getCollectionRequest(resourceType string, queryParams ...[]string) (*http.Request, error) {
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

func (o OctopusClient) GetSpace(resources *octopus2.Space) (funcErr error) {
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

func (o OctopusClient) GetResource(resourceType string, resources any) (exists bool, funcErr error) {
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

	if res.StatusCode == 404 {
		return false, nil
	}

	if res.StatusCode != 200 {
		return false, errors.New("did not find the requested resource: " + resourceType)
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

func (o OctopusClient) GetResourceById(resourceType string, id string, resources any) (exists bool, funcErr error) {
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

func (o OctopusClient) GetAllResources(resourceType string, resources any, queryParams ...[]string) (funcErr error) {
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
