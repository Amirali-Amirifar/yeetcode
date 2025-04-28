package scheduler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"gorm.io/gorm"
)

const (
	timeoutSeconds = 600
	maxRetries     = 2
)

// AssignPendingSubmissions continuously polls pending submissions and processes them
func AssignPendingSubmissions(database *gorm.DB) {
	// Step 1: Reset stuck submissions (where status is 'in_progress') to 'pending'
	resetStuckSubmissions(database)

	for {
		// Step 2: Query the database for a pending submission
		var submission db.Submission
		err := database.Transaction(func(tx *gorm.DB) error {
			result := tx.Raw(`
                SELECT *
                FROM submissions
                WHERE status = ?
                ORDER BY created_at ASC
                LIMIT 1
                FOR UPDATE SKIP LOCKED`, StatusPending).Scan(&submission)

			if result.Error != nil || result.RowsAffected == 0 {
				// No pending submissions, return and wait
				return nil
			}

			submission.Status = string(StatusInProgress)
			now := time.Now()
			submission.LastAssignedAt = &now
			return tx.Save(&submission).Error
		})

		if err != nil {
			log.Println("Error while assigning pending submission:", err)
		}

		if submission.Id != 0 {
			processSubmission(database, &submission)
		}

		time.Sleep(2 * time.Second)
	}
}

func resetStuckSubmissions(database *gorm.DB) {
	err := database.Model(&db.Submission{}).Where("status = ?", StatusInProgress).Updates(map[string]interface{}{
		"status": StatusPending,
	}).Error

	if err != nil {
		log.Println("Error resetting stuck submissions to pending:", err)
		return
	}

	log.Println("Successfully reset all stuck submissions to pending.")
}

// Process a single submission by calling the code_runner API
func processSubmission(database *gorm.DB, submission *db.Submission) {
	var problem db.Question
	if err := database.First(&problem, submission.QuestionId).Error; err != nil {
		log.Printf("Failed to fetch problem for submission ID %d: %v", submission.Id, err)
		updateSubmissionStatus(database, submission.Id, StatusInternalError, "", "Problem data not found")
		return
	}
	timeLimit := problem.TimeLimit
	memoryLimit := problem.MemoryLimit

	var testCases []db.TestCase
	if err := database.Where("question_id = ?", submission.QuestionId).Find(&testCases).Error; err != nil {
		log.Printf("Failed to fetch test cases for problem ID %d: %v", submission.QuestionId, err)
		updateSubmissionStatus(database, submission.Id, StatusInternalError, "", "Failed to fetch test cases")
		return
	}
	inputs := make([]string, len(testCases))
	outputs := make([]string, len(testCases))
	for i, tc := range testCases {
		inputs[i] = tc.Input
		outputs[i] = tc.Output
	}

	payload := map[string]interface{}{
		"code":         submission.Code,
		"inputs":       inputs,
		"outputs":      outputs,
		"time_limit":   timeLimit,
		"memory_limit": memoryLimit,
	}
	payloadBytes, _ := json.Marshal(payload)

	result, err := makeCodeRunnerRequest(payloadBytes)
	if err != nil {
		log.Printf("Code runner request failed for submission ID %d: %v", submission.Id, err)
		if submission.RetryCount >= maxRetries {
			updateSubmissionStatus(database, submission.Id, StatusTimeout, "", "Code runner timeout")
		} else {
			submission.RetryCount++
			updateSubmissionStatus(database, submission.Id, StatusPending, "", "")
		}
		return
	}

	casesRaw, ok := result["cases"]
	if !ok {
		log.Printf("Response from code_runner for submission ID %d missing 'cases': %v", submission.Id, result)
		updateSubmissionStatus(database, submission.Id, StatusInternalError, "", "Invalid response: no cases")
		return
	}

	testCasesRes, ok := casesRaw.([]interface{})
	if !ok {
		log.Printf("Invalid cases array in code_runner response for submission ID %d: %v", submission.Id, result)
		updateSubmissionStatus(database, submission.Id, StatusInternalError, "", "Invalid response: cases not array")
		return
	}

	overallStatus := StatusAccepted
	var errorMessages []string

	for i, c := range testCasesRes {
		caseMap, ok := c.(map[string]interface{})
		if !ok {
			overallStatus = StatusInternalError
			errorMessages = append(errorMessages, fmt.Sprintf("Malformed test case result at #%d", i+1))
			continue
		}
		status, _ := caseMap["status"].(string)
		errorMsg, _ := caseMap["error"].(string)
		switch status {
		case "OK":
			continue
		case "Wrong Answer":
			if overallStatus == StatusAccepted { // First failure takes status down
				overallStatus = StatusWrongAnswer
			}
			errorMessages = append(errorMessages, fmt.Sprintf("Test case %d: Wrong Answer", i+1))
		case "Compile Error":
			overallStatus = StatusCompileError
			errorMessages = append(errorMessages, errorMsg)
			break
		case "Runtime Error":
			if overallStatus == StatusAccepted || overallStatus == StatusWrongAnswer {
				overallStatus = StatusRuntimeError
			}
			errorMessages = append(errorMessages, fmt.Sprintf("Test case %d: Runtime Error %s", i+1, errorMsg))
		case "Time Limit Exceeded":
			if overallStatus == StatusAccepted || overallStatus == StatusWrongAnswer || overallStatus == StatusRuntimeError {
				overallStatus = StatusTimeLimitExceeded
			}
			errorMessages = append(errorMessages, fmt.Sprintf("Test case %d: Time Limit Exceeded", i+1))
		case "Memory Limit Exceeded":
			overallStatus = StatusMemoryLimitExceeded
			errorMessages = append(errorMessages, fmt.Sprintf("Test case %d: Memory Limit Exceeded", i+1))
		case "InternalError", "Internal Error":
			overallStatus = StatusInternalError
			errorMessages = append(errorMessages, fmt.Sprintf("Test case %d: Internal Error: %s", i+1, errorMsg))
			break
		default:
			overallStatus = StatusInternalError
			errorMessages = append(errorMessages, fmt.Sprintf("Test case %d: Unknown status: %s", i+1, status))
			break
		}
		if status == "Compile Error" || status == "InternalError" || status == "Internal Error" {
			break
		}
	}

	switch overallStatus {
	case StatusAccepted:
		updateSubmissionStatus(database, submission.Id, StatusAccepted, "", "")
	case StatusWrongAnswer:
		updateSubmissionStatus(database, submission.Id, StatusWrongAnswer, "", strings.Join(errorMessages, "; "))
	case StatusCompileError:
		updateSubmissionStatus(database, submission.Id, StatusCompileError, "", strings.Join(errorMessages, "; "))
	case StatusRuntimeError:
		updateSubmissionStatus(database, submission.Id, StatusRuntimeError, "", strings.Join(errorMessages, "; "))
	case StatusTimeLimitExceeded:
		updateSubmissionStatus(database, submission.Id, StatusTimeLimitExceeded, "", strings.Join(errorMessages, "; "))
	case StatusMemoryLimitExceeded:
		updateSubmissionStatus(database, submission.Id, StatusMemoryLimitExceeded, "", strings.Join(errorMessages, "; "))
	case StatusTimeout:
		updateSubmissionStatus(database, submission.Id, StatusTimeout, "", strings.Join(errorMessages, "; "))
	default:
		updateSubmissionStatus(database, submission.Id, StatusInternalError, "", strings.Join(errorMessages, "; "))
	}
}

func makeCodeRunnerRequest(payload []byte) (map[string]interface{}, error) {
	codeRunnerURL := "http://localhost:8084/run"
	req, err := http.NewRequest("POST", codeRunnerURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeoutSeconds * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("code_runner returned a non-200 response")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func updateSubmissionStatus(database *gorm.DB, submissionID uint, status SubmissionStatus, output, errorMsg string) {
	database.Model(&db.Submission{}).Where("id = ?", submissionID).Updates(map[string]interface{}{
		"status":       status,
		"output":       output,
		"error":        errorMsg,
		"processed_at": time.Now(),
	})
}
