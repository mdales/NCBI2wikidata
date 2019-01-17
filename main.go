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
	"bufio"
	//"encoding/json"
	"fmt"
	//"log"
	"os"
	"strings"
)

var CC_LICENSE_ITEM_IDS = map[string]string{
	"CC0":         "Q6938433",
	"CC BY":       "Q6905323",
	"CC BY-NC-ND": "Q6937225",
	"CC BY-NC":    "Q6936496",

	// These aren't in the NCBI OA list, but we might get them later from
	// the EuroPMC API
	"CC BY 2.5": "Q18810333",
	"CC BY 4.0": "Q20007257",
}

// Load the NCBI open access file list so we can map PMID -> Copyright
//
// First line is date file was generated, rest are tab separated info on papers. Example:
// oa_package/87/30/PMC17774.tar.gz	Arthritis Res. 1999 Oct 14; 1(1):63-70	PMC17774	PMID:11056661	NO-CC CODE
//
func LoadLicenses(filename string) (map[string]string, error) {

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	lookup := make(map[string]string)

	var line string
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			continue
		}

		pmid := parts[3]
		license := strings.TrimSuffix(parts[4], "\n")

		split_pmid := strings.Split(pmid, ":")
		if len(split_pmid) == 2 {
			// Not all entries have a PMID, but this one did
			lookup[split_pmid[1]] = license
		}

		// Also add in the PMCID
		if len(parts[2]) > 0 {
			lookup[parts[2]] = license
		} else {
			fmt.Printf("%s", line)
		}
	}

	return lookup, nil
}

type Record struct {
	Title           string `json:"P1476"`
	MainSubjects    []MeshDescriptorName `json:"P921"`
	PublicationType string `json:"P31"`
	//Publisher       string `json:"P123"`
	PublicationDate string `json:"P577"`
	Publication     string `json:"P1433"`
	ISSN            string `json:"hmm"`
	License         string `json:"P275"`
	PMID            string `json:"P698"`
	PMCID           string `json:"P932"`
}

func main() {
	fmt.Println("Hello, world")

	license_lookup, err := LoadLicenses("oa_file_list.txt")
	if err != nil {
		panic(err)
	}

	search_request := ESearchRequest{
		DB:         "pubmed",
		APIKey:     os.Getenv("NCBI_API_KEY"),
		Term:       "\"Rett Syndrome\"[Mesh Major Topic] AND Review[ptyp]",
		RetMax:     5,
		UseHistory: true,
	}

	search_resp, rerr := search_request.Do()
	if rerr != nil {
		panic(rerr)
	}

	fmt.Printf("Search returned %d of %s matches.\n", len(search_resp.IDs), search_resp.Count)

	fetch_request := EFetchHistoryRequest{
		DB:       "pubmed",
		WebEnv:   search_resp.WebEnv,
		QueryKey: search_resp.QueryKey,
		APIKey:   os.Getenv("NCBI_API_KEY"),
	}

	fetch_resp, ferr := fetch_request.Do()
	if ferr != nil {
		panic(ferr)
	}

	fmt.Printf("Fetched %d articles.\n", len(fetch_resp.Articles))

	records := make([]Record, 0)


	// Query wikidata for some information
	pmcid_list := make([]string, 0)
	issn_list := make([]string, 0)
	main_subject_list := make([]string, 0)

	for _, article := range fetch_resp.Articles {

		// Is there a PMCID for this paper
		var pmcid string
		for _, articleID := range article.PubMedData.ArticleIDList.ArticleIDs {
			if articleID.IDType == "pmc" {
				pmcid = strings.TrimPrefix(articleID.ID, "PMC")
				break
			}
		}
		if pmcid != "" {
			pmcid_list = append(pmcid_list, pmcid)
		}

		var license string
		if l, ok := license_lookup[article.MedlineCitation.PMID]; ok {
			license = l
		}
		if license == "" {
			// didn't find one with PMID, try PMCID
			if l, ok := license_lookup[pmcid]; ok {
				license = l
			}
		}

		if license == "" {
			continue
		}

		subjects := make([]MeshDescriptorName, 0)
		for _, mesh := range article.MedlineCitation.MeshHeadingList.MeshHeadings {
			major := mesh.DescriptorName.MajorTopicYN == "Y"
			for _, qual := range mesh.QualifierNames {
				major = major || qual.MajorTopicYN == "Y"
			}
			if major {
			    main_subject_list = append(main_subject_list, mesh.DescriptorName.MeshID)
				subjects = append(subjects, mesh.DescriptorName)
			}
		}

		p := article.MedlineCitation.Article[0].Journal.JournalIssue.PubDate
		pubdate := fmt.Sprintf("%s-%d", p.Month, p.Year)
		if p.Year == 0 || p.Month == "" {
			p := article.MedlineCitation.Article[0].ArticleDate
			pubdate = fmt.Sprintf("%d-%d", p.Month, p.Year)
		}

		var pubtype string
		if len(article.MedlineCitation.Article[0].PublicationTypeList.PublicationTypes) > 0 {
			pubtype = article.MedlineCitation.Article[0].PublicationTypeList.PublicationTypes[0].Type
		}

		issn := article.MedlineCitation.Article[0].Journal.ISSN
		if issn != "" {
			issn_list = append(issn_list, issn)
		}

		r := Record{
			Title:           article.MedlineCitation.Article[0].ArticleTitle,
			PMID:            article.MedlineCitation.PMID,
			PMCID:           pmcid,
			License:         license,
			MainSubjects:    subjects,
			PublicationDate: pubdate,
			Publication:     article.MedlineCitation.Article[0].Journal.Title,
			ISSN:            issn,
			PublicationType: pubtype,
		}

		records = append(records, r)
	}

	fmt.Printf("We got information on %d records.\n", len(records))

	pmcid_wikidata_items, perr := PMCIDsToWDItem(pmcid_list)
	if perr != nil {
		panic(perr)
	}
	issn_wikidata_items, ierr := ISSNsToWDItem(issn_list)
	if ierr != nil {
		panic(ierr)
	}
	drug_wikidata_items, d1err := DrugsToWDItem(main_subject_list)
	if d1err != nil {
	    panic (d1err)
	}
	disease_wikidata_items, d2err := DiseasesToWDItem(main_subject_list)
	if d2err != nil {
	    panic (d2err)
	}

	f, e := os.Create("results.csv")
	if e != nil {
		panic(e)
	}
	defer f.Close()
	f.WriteString("Title\tItem\tPMID\tPMCID\tLicense\tLicense Item\tMain Subjects\tPublication Date\tPublication\tISSN\tISSN item\tPublication Type\n")
	for _, record := range records {

		item := pmcid_wikidata_items[record.PMCID]
		issn_item := issn_wikidata_items[record.ISSN]

		main_subjects := ""
		for idx, subject := range record.MainSubjects {
		    if idx != 0 {
		        main_subjects += "; "
		    }
		    main_subjects += subject.Name
		    l := drug_wikidata_items[subject.MeshID]
		    if disease_wikidata_items[subject.MeshID] != "" {
		        if l != "" {
		            l += ", "
		        }
		        l += disease_wikidata_items[subject.MeshID]
		    }
            if l != "" {
                main_subjects += fmt.Sprintf(" (%s)", l)
            }
		}

		f.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			record.Title, item, record.PMID, record.PMCID, record.License,
			CC_LICENSE_ITEM_IDS[record.License], main_subjects,
			record.PublicationDate, record.Publication, record.ISSN, issn_item, record.PublicationType))
	}
}
