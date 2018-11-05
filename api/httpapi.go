/*
 * go4api - a api testing tool written in Go
 * Created by: Ping Zhu 2018
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package api

import (
    // "fmt"
    "time"
    "sync"
    // "reflect"
    "net/http"  
    "strings"
    // "encoding/json"

    "go4api/cmd"
    "go4api/lib/testcase"                                                                                                                             
    "go4api/assertion"
    "go4api/protocal/http"
    // "go4api/sql"
)


func HttpApi(wg *sync.WaitGroup, resultsExeChan chan testcase.TestCaseExecutionInfo, baseUrl string, oTcData testcase.TestCaseDataInfo) {
    //
    defer wg.Done()
    //
    start_time := time.Now()
    start_str := start_time.Format("2006-01-02 15:04:05.999999999")
    //--- TBD: here to identify and call the builtin functions in Body, then modify the tcData
    tcData := oTcData
    if !cmd.Opt.IfMutation {
        tcData = EvaluateBuiltinFunctions(oTcData)
    }
    // setUp
    tcSetUpResult := RunTcSetUp(tcData)
    //
    var actualStatusCode int
    var actualHeader http.Header
    var actualBody []byte
    var httpResult string
    var TestMessages []*testcase.TestMessage
    if IfValidHttp(tcData) == true {
        expStatus := tcData.TestCase.RespStatus()
        expHeader := tcData.TestCase.RespHeaders()
        expBody := tcData.TestCase.RespBody()
        //
        actualStatusCode, actualHeader, actualBody = CallHttp(baseUrl, tcData)
        // (3). compare
        tcName := tcData.TcName()
        httpResult, TestMessages = Compare(tcName, actualStatusCode, actualHeader, actualBody, expStatus, expHeader, expBody)

        // (4). here to generate the outputs file if the Json has "outputs" field
        WriteOutputsDataToFile(httpResult, tcData, actualStatusCode, actualHeader, actualBody)
        WriteOutEnvVariables(httpResult, tcData, actualStatusCode, actualHeader, actualBody)
        WriteSession(httpResult, tcData, actualStatusCode, actualHeader, actualBody)
    } else {
        httpResult = "NoHttp"
        actualStatusCode = 999
    }
    // tearDown
    tcTearDownResult := RunTcTearDown(tcData)

    end_time := time.Now()
    end_str := end_time.Format("2006-01-02 15:04:05.999999999")

    testResult := "Success"
    if tcSetUpResult == "Fail" || httpResult == "Fail" || tcTearDownResult == "Fail" {
        testResult = "Fail"
    }

    // get the TestCaseExecutionInfo
    tcExecution := testcase.TestCaseExecutionInfo {
        TestCaseDataInfo: &tcData,
        SetUpResult: tcSetUpResult,
        HttpResult: httpResult,
        ActualStatusCode: actualStatusCode,
        StartTime: start_str,
        EndTime: end_str,
        TestMessages: TestMessages,
        StartTimeUnixNano: start_time.UnixNano(),
        EndTimeUnixNano: end_time.UnixNano(),
        DurationUnixNano: end_time.UnixNano() - start_time.UnixNano(),
        ActualBody: actualBody,
        TearDownResult: tcTearDownResult,
        TestResult: testResult,
    }

    // (6). write the channel to executor for scheduler and log
    resultsExeChan <- tcExecution
}

func IfValidHttp (tcData testcase.TestCaseDataInfo) bool {
    ifValidHttp := true

    if tcData.TestCase.Request() == nil {
        ifValidHttp = false
    }

    return ifValidHttp
}


func CallHttp(baseUrl string, tcData testcase.TestCaseDataInfo) (int, http.Header, []byte) {
    // urlStr := tcData.TestCase.UrlRaw(baseUrl)
    urlStr := tcData.TestCase.UrlEncode(baseUrl)
    //
    apiMethodSelector, apiMethod, bodyText, bodyMultipart, boundary := GetPayloadInfo(tcData)
    //
    reqHeaders := make(map[string]interface{})
    reqHeaders = tcData.TestCase.ReqHeaders()
    // set the boundary to headers, if multipart
    if boundary != "" {
        reqHeaders["Content-Type"] = boundary
    }

    // < !! ----------- !! >
    // (1). Actual response
    var actualStatusCode int
    var actualHeader http.Header
    var actualBody []byte
    // 
    httpRequest := protocal.HttpRestful{}
    if apiMethodSelector == "POSTMultipart" {
        actualStatusCode, actualHeader, actualBody = httpRequest.Request(urlStr, apiMethod, reqHeaders, bodyMultipart)    
    } else {
        actualStatusCode, actualHeader, actualBody = httpRequest.Request(urlStr, apiMethod, reqHeaders, bodyText)
        }

    return actualStatusCode, actualHeader, actualBody
}


func Compare(tcName string, actualStatusCode int, actualHeader http.Header, actualBody []byte, 
        expStatus map[string]interface{}, expHeader map[string]interface{}, expBody map[string]interface{}) (string, []*testcase.TestMessage) {
    //
    var testResults []bool
    var testMessages []*testcase.TestMessage
    // status
    testResultsS, testMessagesS := CompareStatus(actualStatusCode, expStatus)
    testResults = append(testResults, testResultsS[0:]...)
    testMessages = append(testMessages, testMessagesS[0:]...)
    // headers
    testResultsH, testMessagesH := CompareHeaders(actualHeader, expHeader)
    testResults = append(testResults, testResultsH[0:]...)
    testMessages = append(testMessages, testMessagesH[0:]...)
    // body
    testResultsB, testMessagesB := CompareBody(actualBody, expBody)
    testResults = append(testResults, testResultsB[0:]...)
    testMessages = append(testMessages, testMessagesB[0:]...)

    // default finalResults
    finalResults := "Success"

    for key := range testResults {
        if testResults[key] == false {
            finalResults = "Fail"
            break
        }
    }
    // testMessagesJson, _ := json.Marshal(testMessages)
    // testMessagesJsonStr := string(testMessagesJson)
    
    return finalResults, testMessages
} 


func CompareStatus(actualStatusCode int, expStatus map[string]interface{}) ([]bool, []*testcase.TestMessage) {
    var testResults []bool
    var testMessages []*testcase.TestMessage
    // status
    for assertionKey, expValue := range expStatus {
        actualValue := actualStatusCode
        key := "status"

        testRes, msg := compareCommon("Status", key, assertionKey, actualValue, expValue)
        
        testMessages = append(testMessages, msg)
        testResults = append(testResults, testRes)
    }

    return testResults, testMessages
} 

func CompareHeaders(actualHeader http.Header, expHeader map[string]interface{}) ([]bool, []*testcase.TestMessage) {
    var testResults []bool
    var testMessages []*testcase.TestMessage
    // headers
    for key, value := range expHeader {
        expHeader_sub := value.(map[string]interface{})
        //
        for assertionKey, expValue := range expHeader_sub {
            // as the http.Header has structure, so that here need to assert if the expValue in []string
            actualValue := strings.Join(actualHeader[key], ",")

            testRes, msg := compareCommon("Headers", key, assertionKey, actualValue, expValue)

            testMessages = append(testMessages, msg)
            testResults = append(testResults, testRes)
        } 
    }

    return testResults, testMessages
} 

func CompareBody(actualBody []byte, expBody map[string]interface{}) ([]bool, []*testcase.TestMessage) {
    var testResults []bool
    var testMessages []*testcase.TestMessage
    // body
    for key, value := range expBody {
        expBody_sub := value.(map[string]interface{})

        for assertionKey, expValue := range expBody_sub {
            // if path, then value - value, otherwise, key - value
            actualValue := GetActualValueByJsonPath(key, actualBody)
            
            testRes, msg := compareCommon("Body", key, assertionKey, actualValue, expValue)

            testMessages = append(testMessages, msg)
            testResults = append(testResults, testRes)
        }
    }

    return testResults, testMessages
} 

func compareCommon (reponsePart string, key string, assertionKey string, actualValue interface{}, expValue interface{}) (bool, *testcase.TestMessage) {
    // Note: As get Go nil, for JSON null, need special care, two possibilities:
    // p1: expResult -> null, but can not find out actualValue, go set it to nil, i.e. null (assertion -> false)
    // p2: expResult -> null, actualValue can be founc, and its value --> null (assertion -> true)
    // but here can not distinguish them
    assertionResults := ""
    var testRes bool

    if actualValue == nil || expValue == nil {
        // if only one nil
        if actualValue != nil || expValue != nil {
            assertionResults = "Failed"
            testRes = false
        // both nil
        } else {
            assertionResults = "Success"
            testRes = true
        }
    // no nil
    } else {
        // call the assertion function
        testResult := assertion.CallAssertion(assertionKey, actualValue, expValue)
        // fmt.Println("--->", key, assertionKey, actualValue, expValue, reflect.TypeOf(actualValue), reflect.TypeOf(expValue), testResult)
        if testResult == false {
            assertionResults = "Failed"
            testRes = false
        } else {
            assertionResults = "Success"
            testRes = true
        }
    }
    //
    msg := testcase.TestMessage {
        AssertionResults: assertionResults,
        ReponsePart: reponsePart,
        FieldName: key,
        AssertionKey:  assertionKey,
        ActualValue: actualValue,
        ExpValue: expValue,   
    }

    return testRes, &msg
}





