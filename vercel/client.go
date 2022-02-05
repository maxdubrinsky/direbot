package vercel

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (vc VercelClient) GetDomainRecords() (*RecordResponse, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.vercel.com/v4/domains/mostadequate.gg/records", nil)
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

func (vc VercelClient) CreateDomainTXTRecord(value string) error {
	record := DnsRecord{
		Name:  "",
		Type:  "TXT",
		Value: value,
	}

	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.vercel.com/v4/domains/mostadequate.gg/records", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+vc.Token)

	_, err = http.DefaultClient.Do(req)

	return err
}
