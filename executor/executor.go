/*
 * go4api - a api testing tool written in Go
 * Created by: Ping Zhu 2018
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package executor

import (
    "fmt"
    "time"
    "os"
    "sort"
    "sync"
    "go4api/types" 
    "go4api/ui"     
    "go4api/ui/js"  
    "go4api/ui/style"                                                                                                                                
    "go4api/utils"
    "go4api/utils/texttmpl"
    // "go4api/api"
    "path/filepath"
    "strings"
    "io/ioutil"
    "strconv"
    "go4api/logger"
    // simplejson "github.com/bitly/go-simplejson"
)


func Scheduler(ch chan int, pStart_time time.Time, options map[string]string) { //client
    if options["ifScenario"] == "" {
        Run(ch, pStart_time, options)
    } else {
        // <!!--> Note: there are two kinds of test cases dependency:
        // type 1. the parent and child has only execution dependency, no data exchange
        // type 2. the parent and child has execution dependency and data exchange dynamically
        // for type 1, the json is rendered by data tables first, then build the tcTree
        // for type 2, build the cases hierarchy first, then render the child cases using the parent's outputs
        //
        RunScenario(ch, pStart_time, options)
        fmt.Println("--")
    }
}

func GetBaseUrl(options map[string]string) string {
    testenv := options["testEnv"]
    baseUrl := ""
    if options["baseUrl"] != "" {
        baseUrl = options["baseUrl"]
    } else {
        _, err := os.Stat(options["testhome"] + "/testconfig/testconfig.json")
        // fmt.Println("err: ", err)
        if err == nil {
            baseUrl = utils.GetBaseUrlFromConfig(options["testhome"] + "/testconfig/testconfig.json", testenv) 
        }
    }
    if baseUrl == "" {
        fmt.Println("Warning: baseUrl is not set")
    } else {
        fmt.Println("baseUrl set to: " + baseUrl)
    }

    return baseUrl
}

func Run(ch chan int, pStart_time time.Time, options map[string]string) { //client
    baseUrl := GetBaseUrl(options)
    // get results dir
    pStart := pStart_time.String()
    resultsDir := GetResultsDir(pStart, options)
    //
    // (1), get the text path, default is ../data/*, then search all the sub-folder to get the test scripts
    //
    tcArray := GetTcArray(options)
    // to check the tcArray, if the case not distinct, report it to fix
    if len(tcArray) != len(GetTcNameSet(tcArray)) {
        fmt.Println("\n!! There are duplicated test case names, please make them distinct\n")
        os.Exit(1)
    }
    //
    // fmt.Println("tcArray:", tcArray, "\n")
    // myabe there needs a scheduler, for priority 1 (w or w/o dependency) -> priority 2 (w or w/o dependency), ...
    // --
    // How to impliment the case Dependency???
    // Two big categories: 
    // (1) case has No parent Dependency or successor Dependency, which can be scheduled concurrently
    // (2) case has parent Dependency or successor Dependency, which has rules to be scheduled concurrently
    // 
    // need a tree to track and schedule the run dynamiclly, but need a dummy root test case
 
    // dummy root tc => {"root", "0", "0", rooTC, "", "", ""}

    root, _ := BuildTree(tcArray)
    fmt.Println("------------------")
    //
    prioritySet := GetPrioritySet(tcArray)
    classifications := GetTestCasesByPriority(prioritySet, tcArray)
    // Note, before starting execution, needs to sort the priorities_set first by priority
    // Note: here is a bug, as the sort results is 1, 10, 11, 2, 3, etc. => fixed
    prioritySet_Int := utils.ConvertStringArrayToIntArray(prioritySet)
    sort.Ints(prioritySet_Int)
    prioritySet = utils.ConvertIntArrayToStringArray(prioritySet_Int)

    // If need to set the Concurrency MAX?
    // fmt.Println("------------------", root, &root)
    // ShowNodes(root)
    // fmt.Println("------------------", root, &root)
    InitNodesRunResult(root, "Ready")
    // fmt.Println("------------------", root, &root)
    // ShowNodes(root)
    // fmt.Println("------------------", root, &root)

    //
    fmt.Println("\n====> test cases execution starts!\n")
    statusReadyCount = 0
    // init the status count list
    statusCountList = make([][]int, len(prioritySet) + 1)
    for i := range statusCountList {
        statusCountList[i] = make([]int, 5)
    }
    //
    for p_index, priority := range prioritySet {
        tcArrayPriority := classifications[priority]
        fmt.Println("====> Priority " + priority + " starts!")
        
        miniLoop:
        for {
            //
            resultsChan := make(chan types.TcRunResults, len(tcArray))
            var wg sync.WaitGroup
            //
            ScheduleNodes(root, &wg, options, priority, resultsChan, pStart, baseUrl, resultsDir)
            //
            wg.Wait()

            close(resultsChan)

            for tcRunResults := range resultsChan {
                // here can refactor to struct
                tcName := tcRunResults.TcName
                parentTestCase := tcRunResults.ParentTestCase
                testResult := tcRunResults.TestResult
                actualStatusCode := tcRunResults.ActualStatusCode
                jsonFile_Base := tcRunResults.JsonFile_Base
                csvFileBase := tcRunResults.CsvFileBase
                rowCsv := tcRunResults.RowCsv
                start := tcRunResults.Start
                end := tcRunResults.End
                testMessages := tcRunResults.TestMessages
                start_time_UnixNano := tcRunResults.Start_time_UnixNano
                end_time_UnixNano := tcRunResults.End_time_UnixNano
                duration_UnixNano := tcRunResults.Duration_UnixNano
                //
                // (1). tcName, testResult, the search result is saved to *findNode
                SearchNode(&root, tcName)
                // (2). 
                RefreshNodeAndDirectChilrenTcResult(*findNode, testResult, start, end, 
                    testMessages, start_time_UnixNano, end_time_UnixNano)
                // fmt.Println("------------------")
                // (3). <--> for log write to file
                resultReportString1 := priority + "," + tcName + "," + parentTestCase + "," + testResult + "," + actualStatusCode + "," + jsonFile_Base + "," + csvFileBase
                resultReportString2 := "," + rowCsv + "," + start + "," + end + "," + "`" + "d" + "`" + "," + strconv.FormatInt(start_time_UnixNano, 10)
                resultReportString3 := "," + strconv.FormatInt(end_time_UnixNano, 10) + "," +  strconv.FormatInt(duration_UnixNano, 10)
                resultReportString :=  resultReportString1 + resultReportString2 + resultReportString3
                // (4). put the execution log into results
                logger.WriteExecutionResults(resultReportString, pStart, resultsDir)
                // fmt.Println("------!!!------")
            }
            // if tcTree has no node with "Ready" status, break the miniloop
            statusReadyCount = 0
            CollectNodeReadyStatus(root, priority)
            // fmt.Println("------------------ statusReadyCount: ", statusReadyCount)
            if statusReadyCount == 0 {
                break miniLoop
            }
        }
        //
        CollectNodeStatusByPriority(root, p_index, priority)
        // (5). also need to put out the cases which has not been executed (i.e. not Success, Fail)
        for _, tc := range tcNotExecutedList {
            // [casename, priority, parentTestCase, ...], tc, jsonFile, csvFile, row in csv
            if tc[1].(string) == priority {
                resultReportString := tc[1].(string) + "," + tc[0].(string) + "," + tc[2].(string) + "," + "ParentFailed" + ",," + tc[4].(string) + "," + tc[5].(string)
                resultReportString = resultReportString + "," + tc[6].(string) + ",,,,,,"

                logger.WriteExecutionResults(resultReportString, pStart, resultsDir)
                // to console
                fmt.Println(resultReportString)
            }
        }
        
        //
        var successCount = statusCountList[p_index][2]
        var failCount = statusCountList[p_index][3]
        //
        fmt.Println("---------------------------------------------------------------------------")
        fmt.Println("----- Priority " + priority + ": " + strconv.Itoa(len(tcArrayPriority)) + " Cases in template -----")
        fmt.Println("----- Priority " + priority + ": " + strconv.Itoa(statusCountList[p_index][0]) + " Cases put onto tcTree -----")
        fmt.Println("----- Priority " + priority + ": " + strconv.Itoa(statusCountList[p_index][0] - successCount - failCount) + " Cases Skipped (Not Executed, due to Parent Failed) -----")
        fmt.Println("----- Priority " + priority + ": " + strconv.Itoa(successCount + failCount) + " Cases Executed -----")
        fmt.Println("----- Priority " + priority + ": " + strconv.Itoa(successCount) + " Cases Success -----")
        fmt.Println("----- Priority " + priority + ": " + strconv.Itoa(failCount) + " Cases Fail -----")
        fmt.Println("---------------------------------------------------------------------------")

        fmt.Println("====> Priority " + priority + " ended! \n")
        // sleep for debug
        time.Sleep(500 * time.Millisecond)
    }
    // ShowNodes(root)
    CollectOverallNodeStatus(root, len(prioritySet))
    // fmt.Println("====> statusCountList final: ", statusCountList)
    //
    var successCount = statusCountList[len(prioritySet)][2]
    var failCount = statusCountList[len(prioritySet)][3]
    //
    fmt.Println("---------------------------------------------------------------------------")
    fmt.Println("----- Total " + strconv.Itoa(len(tcArray)) + " Cases in template -----")
    fmt.Println("----- Total " + strconv.Itoa(statusCountList[len(prioritySet)][0]) + " Cases put onto tcTree -----")
    fmt.Println("----- Total " + strconv.Itoa(statusCountList[len(prioritySet)][0] - successCount - failCount) + " Cases Skipped (Not Executed, due to Parent Failed) -----")
    fmt.Println("----- Total " + strconv.Itoa(successCount + failCount) + " Cases Executed -----")
    fmt.Println("----- Total " + strconv.Itoa(successCount) + " Cases Success -----")
    fmt.Println("----- Total " + strconv.Itoa(failCount) + " Cases Fail -----")
    fmt.Println("---------------------------------------------------------------------------\n\n")


    // generate the html report based on template, and results data
    // time.Sleep(1 * time.Second)
    pEnd_time := time.Now()
    //
    GenerateTestReport(resultsDir, pStart_time, pStart, pEnd_time)
    //
    fmt.Println("Report Generated at: " + resultsDir + "index.html")
    fmt.Println("Execution Finished at: " + pEnd_time.String())

    // channel code, can be used for the overall success or fail indicator, especially for CI/CD
    ch <- 1

}


func GetTcArray(options map[string]string) [][]interface{} {
    var tcArray [][]interface{}

    jsonFileList, _ := utils.WalkPath(options["testhome"] + "/testdata/", ".json")
    // fmt.Println("jsonFileList:", jsonFileList, "\n")
    // to ge the json and related data file, then get tc from them
    for _, jsonFile := range jsonFileList {
        csvFileList := GetCsvDataFilesForJsonFile(jsonFile, "_dt")
        // to get the json test data directly (if not template) based on template (if template)
        // tcInfos: [[casename, priority, parentTestCase, ], ...]
        var tcInfos [][]interface{}
        if len(csvFileList) > 0 {
            tcInfos = ConstructTcInfosBasedOnJsonTemplateAndDataTables(jsonFile, csvFileList)
        } else {
            tcInfos = ConstructTcInfosBasedOnJson(jsonFile)
        }

        // fmt.Println("tcInfos:", tcInfos, "\n")
        
        for _, tc := range tcInfos {
            tcArray = append(tcArray, tc)
        }
    }

    return tcArray
}

func GetCsvDataFilesForJsonFile(jsonFile string, suffix string) []string {
    // here search out the csv files under the same dir, not to use utils.WalkPath as it is recursively
    var csvFileListTemp []string
    infos, err := ioutil.ReadDir(filepath.Dir(jsonFile))
    if err != nil {
      panic(err)
    }

    // get the csv file, ignore the fields "inputs", "outputs"
    for _, info := range infos {
      if filepath.Ext(info.Name()) == ".csv" {
        csvFileListTemp = append(csvFileListTemp, filepath.Join(filepath.Dir(jsonFile), info.Name()))
      }
    }
    // 
    var csvFileList []string
    for _, csvFile := range csvFileListTemp {
        csvFileName := strings.TrimRight(filepath.Base(csvFile), ".csv")
        jsonFileName := strings.TrimRight(filepath.Base(jsonFile), ".json")
        // Note: the json file realted data table files is pattern: jsonFileName + "_dt[*]"
        
        // if             
        if strings.Contains(csvFileName, jsonFileName + suffix) {
            csvFileList = append(csvFileList, csvFile)
        }
    }

    return csvFileList
}


func ConstructTcInfosBasedOnJsonTemplateAndDataTables(jsonFile string, csvFileList []string) [][]interface{} {
    var tcInfos [][]interface{}

    for _, csvFile := range csvFileList {
        csvRows := utils.GetCsvFromFile(csvFile)
        for i, csvRow := range csvRows {
            // starting with data row
            if i > 0 {
                // outTempFile := texttmpl.GenerateJsonFileBasedOnTemplateAndCsv(jsonFile, csvRows[0], csvRow, tmpJsonDir)
                // tcJsonsTemp := utils.GetTestCaseJsonFromTestDataFile(outTempFile)
                outTempJson := texttmpl.GenerateJsonBasedOnTemplateAndCsv(jsonFile, csvRows[0], csvRow)
                tcJsonsTemp := utils.GetTestCaseJsonFromTestData(outTempJson)
                // as the json is generated based on templated dynamically, so that, to cache all the resulted json in array
                var tcInfo []interface{}
                for _, tc := range tcJsonsTemp {
                    // to get the case info like [casename, priority, parentTestCase, ...], tc, jsonFile, csvFile, row in csv
                    // Note: row in csv = i + 1 (i.e. plus csv header line)
                    tcInfo = utils.GetTestCaseBasicInfoFromTestData(tc)
                    tcInfo = append(tcInfo, tc, jsonFile, csvFile, strconv.Itoa(i + 1))
                    tcInfos = append(tcInfos, tcInfo)
                }
            }
        }
    }
    return tcInfos
}

func ConstructTcInfosBasedOnJson(jsonFile string) [][]interface{} {
    var tcInfos [][]interface{}

    csvFile := ""
    csvRow := ""
    // outTempFile := texttmpl.GenerateJsonFileBasedOnTemplateAndCsv(jsonFile, []string{""}, []string{""}, tmpJsonDir)
    // tcJsonsTemp := utils.GetTestCaseJsonFromTestDataFile(outTempFile)
    outTempJson := texttmpl.GenerateJsonBasedOnTemplateAndCsv(jsonFile, []string{""}, []string{""})
    tcJsonsTemp := utils.GetTestCaseJsonFromTestData(outTempJson)
    // as the json is generated based on templated dynamically, so that, to cache all the resulted json in array
    var tcInfo []interface{}
    for _, tc := range tcJsonsTemp {
        // to get the case info like [casename, priority, parentTestCase, ...]
        tcInfo = utils.GetTestCaseBasicInfoFromTestData(tc)
        tcInfo = append(tcInfo, tc, jsonFile, csvFile, csvRow)
        tcInfos = append(tcInfos, tcInfo)
    }

    return tcInfos
}


func GetPrioritySet(tcArray [][]interface{}) []string {
    // get the priorities
    var priorities []interface{}
    for _, tc := range tcArray {
        priorities = append(priorities, tc[1])
    }
    // go get the distinct key in priorities
    keys := make(map[string]bool)
    prioritySet := []string{}
    for _, entry := range priorities {
        // uses 'value, ok := map[key]' to determine if map's key exists, if ok, then true
        if _, value := keys[entry.(string)]; !value {
            keys[entry.(string)] = true
            prioritySet = append(prioritySet, entry.(string))
        }
    }

    return prioritySet
}

func GetTcNameSet(tcArray [][]interface{}) []string {
    // get the tcNames
    var tcNames []interface{}
    for _, tc := range tcArray {
        tcNames = append(tcNames, tc[0])
    }
    // go get the distinct key in tcNames
    keys := make(map[string]bool)
    tcNameSet := []string{}
    for _, entry := range tcNames {
        // uses 'value, ok := map[key]' to determine if map's key exists, if ok, then true
        if _, value := keys[entry.(string)]; !value {
            keys[entry.(string)] = true
            tcNameSet = append(tcNameSet, entry.(string))
        }
    }

    return tcNameSet
}

func GetTestCasesByPriority(prioritySet []string, tcArray [][]interface{}) map[string][][]interface{} {
    // build the map
    classifications := make(map[string][][]interface{})
    for _, entry := range prioritySet {
        for _, tc := range tcArray {
            // tc[1] represents the priority
            if entry == tc[1] {
                classifications[entry] = append(classifications[entry], tc)
            }
        }
    }
    // fmt.Println("classifications: ", classifications)
    return classifications
}


func GenerateTestReport(resultsDir string, pStart_time time.Time, pStart string, pEnd_time time.Time) {
    // read the resource under /ui/*
    // fmt.Println("ui: ", ui.Index_template)

    // copy the value of var Index to file
    utils.GenerateFileBasedOnVarOverride(ui.Index, resultsDir + "index.html")

    //
    err := os.MkdirAll(resultsDir + "js", 0777)
    if err != nil {
      panic(err) 
    }
    // copy the value of var js.Js to file
    texttmpl.GenerateHtmlJsCSSFromTemplateAndVar(js.Results, pStart_time, pEnd_time, resultsDir, resultsDir + pStart + ".log")
    //
    utils.GenerateFileBasedOnVarOverride(js.Js, resultsDir + "js/go4api.js")
    //
    err = os.MkdirAll(resultsDir + "style", 0777)
    if err != nil {
      panic(err) 
    }
    // copy the value of var style.Style to file
    utils.GenerateFileBasedOnVarOverride(style.Style, resultsDir + "style/go4api.css")
}

func GetResultsDir(pStart string, options map[string]string) string {
    var resultsDir string
    err := os.MkdirAll(options["testresults"] + "/" + pStart + "/", 0777)
    if err != nil {
      panic(err) 
    } else {
        resultsDir = options["testresults"] + "/" + pStart + "/"
    }

    return resultsDir
}

func GetTmpJsonDir(path string) string {
    // check if the /tmp/go4api_wfasf exists, if exists, then rm first
    os.RemoveAll("/tmp/" + path)
    //
    var resultsDir string
    err := os.Mkdir("/tmp/" + path + "/", 0777)
    if err != nil {
      panic(err) 
    } else {
        resultsDir = "/tmp/" + path + "/"
    }

    return resultsDir
}

