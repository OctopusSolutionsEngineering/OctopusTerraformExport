package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
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

	req.Header.Set("X-Octopus-ApiKey", o.ApiKey)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	}

	return res.StatusCode != 404, nil
}

func (o OctopusClient) lookupSpaceAsName() (string, error) {
	if len(strings.TrimSpace(o.Space)) == 0 {
		return "", errors.New("space can not be empty")
	}

	requestURL := fmt.Sprintf("%s/api/Spaces?take=1000&partialName=%s", o.Url, url.QueryEscape(o.Space))
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return "", err
	}

	req.Header.Set("X-Octopus-ApiKey", o.ApiKey)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", nil
	}
	defer res.Body.Close()

	collection := model.GeneralCollection[model.Space]{}
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

func (o OctopusClient) getSpaceRequest() (*http.Request, error) {
	spaceUrl, err := o.getSpaceUrl()

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, spaceUrl, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Octopus-ApiKey", o.ApiKey)

	return req, nil
}

func (o OctopusClient) getRequest(resourceType string, id string) (*http.Request, error) {
	spaceUrl, err := o.getSpaceUrl()

	if err != nil {
		return nil, err
	}

	requestURL := spaceUrl + "/" + resourceType + "/" + id
	req, err := http.NewRequest(http.MethodPost, requestURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Octopus-ApiKey", o.ApiKey)

	return req, nil
}

func (o OctopusClient) getCollectionRequest(resourceType string) (*http.Request, error) {
	spaceUrl, err := o.getSpaceUrl()

	if err != nil {
		return nil, err
	}

	requestURL := spaceUrl + "/" + resourceType + "?take=10000"
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Octopus-ApiKey", o.ApiKey)

	return req, nil
}

func (o OctopusClient) GetSpace(resources *model.Space) error {
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
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(resources)
}

func (o OctopusClient) GetResourceById(resourceType string, id string, resources any) error {
	req, err := o.getRequest(resourceType, id)

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
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(resources)
}

func (o OctopusClient) GetResourceByName(resourceType string, names string) {

}

func (o OctopusClient) GetAllResources(resourceType string, resources any) error {
	req, err := o.getCollectionRequest(resourceType)

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
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(resources)
}
