package bdd

import (
	"fmt"
	"testing"

	"github.com/cucumber/godog"
	// 若要輸出到 os.Stdout 就 import "os"
	"os"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Paths:  []string{"./featureFiles"}, // 指向 feature 檔相對路徑
			Format: "pretty",
			Output: os.Stdout, // 將結果輸出到終端
		},
	}

	// 若 suite.Run() != 0 表示測試失敗，可以讓 t.Fail() 或 t.Fatal()
	if suite.Run() != 0 {
		t.Fail()
	}
}

// 這個函式用來註冊 Gherkin 與 Step Definition 的對應
func InitializeScenario(s *godog.ScenarioContext) {
	s.Step(`^A user with email "([^"]*)" and password "([^"]*)" exists$`, aUserWithEmailAndPasswordExists)
	s.Step(`^I attempt to login with "([^"]*)" and "([^"]*)"$`, iAttemptToLoginWith)
	s.Step(`^I should get a "([^"]*)" response$`, iShouldGetAResponse)
	s.Step(`^I should receive a valid session token$`, iShouldReceiveAValidSessionToken)
}

// 以下示例 Step function
var inMemoryUsers = map[string]string{}
var lastLoginResult string
var lastSessionToken string

func aUserWithEmailAndPasswordExists(email, password string) error {
	inMemoryUsers[email] = password
	return nil
}

func iAttemptToLoginWith(email, password string) error {
	if inMemoryUsers[email] == password {
		lastLoginResult = "success"
		lastSessionToken = "token123"
	} else {
		lastLoginResult = "failure"
		lastSessionToken = ""
	}
	return nil
}

func iShouldGetAResponse(expected string) error {
	if lastLoginResult != expected {
		return fmt.Errorf("expected %s, but got %s", expected, lastLoginResult)
	}
	return nil
}

func iShouldReceiveAValidSessionToken() error {
	if lastSessionToken == "" {
		return fmt.Errorf("no session token received")
	}
	return nil
}
