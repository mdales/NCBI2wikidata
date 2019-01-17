//   Copyright 2019 Content Mine Ltd
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

const EFETCH_URL string = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi"

type PubmedArticleSet struct {
	XMLName  xml.Name        `xml:"PubmedArticleSet"`
	Articles []PubmedArticle `xml:"PubmedArticle"`
}

type PubmedArticle struct {
	XMLName         xml.Name        `xml:"PubmedArticle"`
	MedlineCitation MedlineCitation `xml:"MedlineCitation"`
	PubMedData      PubMedData      `xml:"PubmedData"`
}

type MedlineCitation struct {
	XMLName                 xml.Name                `xml:"MedlineCitation"`
	Status                  string                  `xml:"Status,attr"`
	Owner                   string                  `xml:"Owner,attr"`
	PMID                    string                  `xml:"PMID"`
	Article                 []Article               `xml:"Article"`
	MeshHeadingList         MeshHeadingList         `xml:"MeshHeadingList"`
	CommentsCorrectionsList CommentsCorrectionsList `xml:"CommentsCorrectionsList"`
}

type Article struct {
	XMLName             xml.Name            `xml:"Article"`
	PubModel            string              `xml:"PubModel,attr"`
	ArticleTitle        string              `xml:"ArticleTitle"`
	Journal             Journal             `xml:"Journal"`
	Language            string              `xml:"Language"`
	PublicationTypeList PublicationTypeList `xml:"PublicationTypeList"`
	ArticleDate         ArticleDate         `xml:"ArticleDate"`
}

type ArticleDate struct {
	XMLName xml.Name `xml:"ArticleDate"`
	Year    int      `xml:"Year"`
	Month   int      `xml:"Month"`
	Day     int      `xml:"Day"`
}

type PublicationTypeList struct {
	XMLName          xml.Name          `xml:"PublicationTypeList"`
	PublicationTypes []PublicationType `xml:"PublicationType"`
}

type PublicationType struct {
	XML  xml.Name `xml:"PublicationType"`
	Type string   `xml:",chardata"`
	UI   string   `xml:"UI,attr"`
}

type Journal struct {
	XMLName         xml.Name     `xml:"Journal"`
	Title           string       `xml:"Title"`
	ISOAbbreviation string       `xml:"ISOAbbreviation"`
	JournalIssue    JournalIssue `xml:"JournalIssue"`
	ISSN            string       `xml:"ISSN"`
}

type JournalIssue struct {
	XMLName    xml.Name `xml:"JournalIssue"`
	CitedMedia string   `xml:"CitedMedia,attr"`
	Volume     string   `xml:"Volume"`
	Issue      string   `xml:"Issue"`
	PubDate    PubDate  `xml:"PubDate"`
}

type PubDate struct {
	XMLName xml.Name `xml:"PubDate"`
	Year    int      `xml:"Year"`
	Month   string   `xml:"Month"`
	Day     int      `xml:"Day"`
}

type MeshHeadingList struct {
	XMLName      xml.Name      `xml:"MeshHeadingList"`
	MeshHeadings []MeshHeading `xml:"MeshHeading"`
}

type MeshHeading struct {
	XMLName        xml.Name            `xml:"MeshHeading"`
	DescriptorName MeshDescriptorName  `xml:"DescriptorName"`
	QualifierNames []MeshQualifierName `xml:"QualifierName"`
}

type MeshDescriptorName struct {
	XMLName      xml.Name `xml:"DescriptorName"`
	Name         string   `xml:",chardata"`
	MeshID       string   `xml:"UI,attr"`
	MajorTopicYN string   `xml:MajorTopicYN,attr"`
}

type MeshQualifierName struct {
	XMLName      xml.Name `xml:"QualifierName"`
	Name         string   `xml:",chardata"`
	MeshID       string   `xml:"UI,attr"`
	MajorTopicYN string   `xml:"MajorTopicYN,attr"`
}

type CommentsCorrectionsList struct {
	XMLName             xml.Name              `xml:"CommentsCorrectionsList"`
	CommentsCorrections []CommentsCorrections `xml:"CommentsCorrections"`
}

type CommentsCorrections struct {
	XMLName   xml.Name `xml:"CommentsCorrections"`
	RefSource string   `xml:"RefSource"`
	PMID      string   `xml:"PMID"`
}

type PubMedData struct {
	XMLName           xml.Name      `xml:"PubmedData"`
	ArticleIDList     ArticleIdList `xml:"ArticleIdList"`
	PublicationStatus string        `xml:"PublicationStatus"`
}

type ArticleIdList struct {
	XMLName    xml.Name    `xml:"ArticleIdList"`
	ArticleIDs []ArticleID `xml:"ArticleId"`
}

type ArticleID struct {
	XMLName xml.Name `xml:"ArticleId"`
	ID      string   `xml:",chardata"`
	IDType  string   `xml:"IdType,attr"`
}

type EFetchHistoryRequest struct {
	DB       string
	WebEnv   string
	QueryKey string
	APIKey   string
}

func (e *EFetchHistoryRequest) Do() (PubmedArticleSet, error) {

	if e.APIKey == "" {
		return PubmedArticleSet{}, fmt.Errorf("No API Key provided.")
	}

	req, err := http.NewRequest("GET", EFETCH_URL, nil)
	if err != nil {
		return PubmedArticleSet{}, err
	}

	q := req.URL.Query()
	q.Add("api_key", e.APIKey)
	q.Add("db", e.DB)
	q.Add("WebEnv", e.WebEnv)
	q.Add("query_key", e.QueryKey)
	q.Add("retmode", "xml")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, response_err := client.Do(req)
	if response_err != nil {
		return PubmedArticleSet{}, response_err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return PubmedArticleSet{}, fmt.Errorf("Status code %d", resp.StatusCode)
		} else {
			return PubmedArticleSet{}, fmt.Errorf("Status code %d: %s", resp.StatusCode, body)
		}
	}

	efetch_resp := PubmedArticleSet{}
	err = xml.NewDecoder(resp.Body).Decode(&efetch_resp)
	if err != nil {
		return PubmedArticleSet{}, err
	}

	return efetch_resp, nil
}
