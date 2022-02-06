package vercel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type VercelClient struct {
	Token string
}

type DnsRecord struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	MxPriority *string `json:"mxPriority,omitempty"`
	Priority   *string `json:"priority,omitempty"`
}

type Record struct {
	DnsRecord
	Id        string `json:"id"`
	Slug      string `json:"slug"`
	Creator   string `json:"creator"`
	Created   *int   `json:"created,omitempty"`
	Updated   *int   `json:"updated,omitempty"`
	CreatedAt *int   `json:"createdAt,omitempty"`
	UpdatedAt *int   `json:"updatedAt,omitempty"`
}

type RecordResponse struct {
	Records []Record `json:"records"`
}

func (vc VercelClient) makeRequest(method string, endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("https://api.vercel.com%s", endpoint)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+vc.Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	return res, err
}

func (vc VercelClient) GetDomainRecords(domain string) (*RecordResponse, error) {
	url := fmt.Sprintf("/v4/domains/%s/records", domain)
	res, err := vc.makeRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data RecordResponse
	err = json.Unmarshal(body, &data)

	return &data, err
}

func (vc VercelClient) CreateDomainTXTRecord(domain, value string) error {
	record := DnsRecord{
		Name:  "",
		Type:  "TXT",
		Value: value,
	}

	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/v4/domains/%s/records", domain)
	_, err = vc.makeRequest(http.MethodPost, url, bytes.NewBuffer(body))

	return err
}
