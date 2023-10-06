package main

import (
	"fmt"
	"strconv"
	"time"
)

var (
	pageSize = 500
)

// Define a function to apply branch detail to the project list
func applyBranchDetail(pl []ProjectSearchList, host, credential string) []ProjectSearchList {
	return branchDetailOfProjects(host, credential, pl)
}

// Define a function to apply owner information to the project list
func applyOwnerInformation(pl []ProjectSearchList, host, credential string) []ProjectSearchList {
	return ownerProject(host, credential, pl)
}

func indexPageNumberCounter(getProjectSearchPage interface{}) int {
	switch page := getProjectSearchPage.(type) {
	case ProjectSearchPage:
		pages := page.Paging.Total / pageSize
		if page.Paging.Total%pageSize > 0 {
			pages++
		}
		if pages < 1 {
			pages = 1
		}
		return pages
	case ProjectSearchOfApplication:
		pages := page.Paging.Total / pageSize
		if page.Paging.Total%pageSize > 0 {
			pages++
		}
		if pages < 1 {
			pages = 1
		}
		return pages
	default:
		return 0
	}

}

func removeByKeys(list []ProjectSearchList, keysToRemove []string) []ProjectSearchList {
	// Create a map to store the keys that need to be removed for efficient lookup.
	keySet := make(map[string]bool)
	for _, key := range keysToRemove {
		keySet[key] = true
	}

	// Create a new list to store the updated items.
	var updatedList []ProjectSearchList

	// Iterate through the original list.
	for _, item := range list {
		// Check if the item's key is not in the set of keys to remove.
		if !keySet[item.Key] {
			// If not, add the item to the updated list.
			updatedList = append(updatedList, item)
		}
	}

	return updatedList
}

func projectLength(host string, credential string) int {
	data := httpRequest(host+projectIndexApi, credential)
	// fmt.Println(string(data))
	var projectSearchPage ProjectSearchPage
	err := dataParse(data, &projectSearchPage)
	if err != nil {
		fmt.Println(err)
	}

	// fmt.Println(projectSearchPage)
	indexPageNumber := indexPageNumberCounter(projectSearchPage)

	return indexPageNumber
}

func listProject(host string, credential string, lengthProject int) []ProjectSearchList {
	dispayJob("list project", "start")
	var projectList []ProjectSearchList
	for pageIndex := 1; pageIndex <= lengthProject; pageIndex++ {
		raw := httpRequest(host+projectScrapeApi+strconv.Itoa(pageIndex),
			credential)
		// fmt.Println(raw)

		var structured ProjectSearchPage

		err := dataParse(raw, &structured)
		// fmt.Println(structured)
		if err != nil {
			fmt.Println(err)
		}

		for projectIndex := range structured.Components {
			projectList = append(projectList, ProjectSearchList{
				Key:  structured.Components[projectIndex].Key,
				Name: structured.Components[projectIndex].Name,
			})
		}
	}
	dispayJob("list project", "end")
	return projectList

}
func findIndexOfLatestDate(dateStrings []string) (int, error) {
	// Common layout for parsing date strings.
	layout := "2006-01-02T15:04:05-0700"

	// Initialize variables to keep track of the latest date and its index.
	var latestDateIndex int
	var latestDate time.Time

	for i, dateStr := range dateStrings {
		parsedDate, err := time.Parse(layout, dateStr)
		if err != nil {
			return -1, err // Return -1 for index and the error.
		}

		// If it's the first valid date or later than the current latestDate, update latestDate and its index.
		if i == 0 || parsedDate.After(latestDate) {
			latestDate = parsedDate
			latestDateIndex = i
		}
	}

	return latestDateIndex, nil
}
func branchDetailOfProjects(host string, credential string, projectList []ProjectSearchList) []ProjectSearchList {
	// fmt.Println("Gather Branch Detail")
	dispayJob("obtain branch data", "start")

	for index := range projectList {
		raw := httpRequest(host+projectBranchesApi+projectList[index].Key, credential)
		var structured ProjectBranchesList
		err := dataParse(raw, &structured)

		handleErr(err)
		var (
			compareNloc     []int
			loc             int
			lastDate        string
			compareLastDate []string
		)
		for branchIndex := range structured.Branches {
			nlocRaw := httpRequest(
				host+ProjectBranchesLocApi+projectList[index].Key+"&branch="+structured.Branches[branchIndex].Name,
				credential)

			var nlocStructured ProjectMeasures

			err := dataParse(nlocRaw, &nlocStructured)
			handleErr(err)

			if len(nlocStructured.Component.Measures) == 0 {
				loc = 0
			} else {
				loc, err = strconv.Atoi(nlocStructured.Component.Measures[0].Value)
				handleErr(err)
			}
			compareNloc = append(compareNloc, loc)

			lastDateRaw := httpRequest(host+ProjectDateAnalysisApi+projectList[index].Key+"&branch="+structured.Branches[branchIndex].Name,
				credential)

			var lastDateStructured ProjectAnalyses
			err = dataParse(lastDateRaw, &lastDateStructured)
			handleErr(err)

			if len(lastDateStructured.Analyses) == 0 {
				lastDate = "0001-01-01T00:00:00+0000"
			} else {
				lastDate = lastDateStructured.Analyses[0].Date
			}

			compareLastDate = append(compareLastDate, lastDate)
		}
		// fmt.Println(projectList[index].Key, compareLastDate)
		branchCalculatedNloc := findIndexOfHighestValue(compareNloc)
		lastAnalysisDate, err := findIndexOfLatestDate(compareLastDate)
		handleErr(err)
		// projectList[index] = ProjectSearchList{
		// 	HighestBranch:      structured.Branches[branchCalculatedNloc].Name,
		// 	Loc:                strconv.Itoa(compareNloc[branchCalculatedNloc]),
		// 	LastAnalysisDate:   compareLastDate[lastAnalysisDate],
		// 	LastAnalysisBranch: structured.Branches[lastAnalysisDate].Name,
		// }
		projectList[index].HighestLinesOfCodeBranch = structured.Branches[branchCalculatedNloc].Name
		projectList[index].LinesOfCode = strconv.Itoa(compareNloc[branchCalculatedNloc])
		projectList[index].LastAnalysisDate = compareLastDate[lastAnalysisDate]
		projectList[index].LastAnalysisBranch = structured.Branches[lastAnalysisDate].Name

	}
	dispayJob("obtain branch data", "end")

	return projectList
}

func ownerProject(host string, credential string, projectList []ProjectSearchList) []ProjectSearchList {
	// fmt.Println("Owner func")
	dispayJob("obtain project owner", "start")

	for index := range projectList {
		raw := httpRequest(host+ProjectUserPermissionsApi+projectList[index].Key, credential)
		var structured ProjectPermissions

		err := dataParse(raw, &structured)
		// fmt.Println(structured)
		handleErr(err)

		// projectList[index] = ProjectSearchList{
		// 	Owner: structured.Users[0].Name,
		// 	Email: structured.Users[0].Email,
		// }
		projectList[index].Owner = structured.Users[0].Name
		projectList[index].Email = structured.Users[0].Email

	}
	dispayJob("obtain project owner", "end")
	return projectList
}
func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
