package bdd

import "github.com/cucumber/godog"

// godog run ./tests/bdd/featureFiles/chat_service.feature                                                                ok | base py | at 00:56:08
// Use of godog CLI is deprecated, please use *testing.T instead.
// See https://github.com/cucumber/godog/discussions/478 for details.
// Feature: 聊天功能
//   In order to communicate effectively
//   As registered users and group admins
//   I want to start private conversations and manage group chats

//   Background:
//     Given "memberA" 已登入並取得 Token "tokenA"
//     And "memberB" 已登入並取得 Token "tokenB"
//     And "adminUser" 已登入並取得 Token "adminToken"
//     And "normalUser" 已登入並取得 Token "userToken"
//     And a 群組聊天室 "Go Club" 已存在 with "adminUser" as Admin and "normalUser" as Member

//   Scenario: 成功建立 1對1 聊天                                                            # ./tests/bdd/featureFiles/chat_service.feature:14
//     When "memberA" 建立 1對1 聊天邀請 "memberB"
//     Then 聊天房間應該包含 "memberA" 和 "memberB"

//   Scenario: 發送與接收訊息                                                                # ./tests/bdd/featureFiles/chat_service.feature:18
//     Given 已存在 1對1 聊天房間 with "memberA" and "memberB"
//     When "memberA" 發送訊息 "Hello B!"
//     Then "memberB" 應該收到訊息 "Hello B!"

//   Scenario: Admin 禁言普通會員                                                           # ./tests/bdd/featureFiles/chat_service.feature:24
//     When "adminUser" 禁言 "normalUser" for 10 分鐘 in "Go Club"
//     Then "normalUser" 無法在 "Go Club" 發送訊息

func StepDefinitioninition1(arg1 string, arg2, arg3 int, arg4 string) error {
	return godog.ErrPending
}

func StepDefinitioninition2(arg1, arg2 string) error {
	return godog.ErrPending
}

func StepDefinitioninition3(arg1, arg2 string) error {
	return godog.ErrPending
}

func StepDefinitioninition4(arg1, arg2 string) error {
	return godog.ErrPending
}

func StepDefinitioninition5(arg1, arg2 string) error {
	return godog.ErrPending
}

func aWithAsAdminAndAsMember(arg1, arg2, arg3 string) error {
	return godog.ErrPending
}

func forIn(arg1, arg2 string, arg3 int, arg4 string) error {
	return godog.ErrPending
}

func token(arg1, arg2 string) error {
	return godog.ErrPending
}

func withAnd(arg1, arg2 int, arg3, arg4 string) error {
	return godog.ErrPending
}

func InitializeChatServiceScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^"([^"]*)" 建立 (\d+)對(\d+) 聊天邀請 "([^"]*)"$`, StepDefinitioninition1)
	ctx.Step(`^聊天房間應該包含 "([^"]*)" 和 "([^"]*)"$`, StepDefinitioninition2)
	ctx.Step(`^"([^"]*)" 發送訊息 "([^"]*)"$`, StepDefinitioninition3)
	ctx.Step(`^"([^"]*)" 應該收到訊息 "([^"]*)"$`, StepDefinitioninition4)
	ctx.Step(`^"([^"]*)" 無法在 "([^"]*)" 發送訊息$`, StepDefinitioninition5)
	ctx.Step(`^a 群組聊天室 "([^"]*)" 已存在 with "([^"]*)" as Admin and "([^"]*)" as Member$`, aWithAsAdminAndAsMember)
	ctx.Step(`^"([^"]*)" 禁言 "([^"]*)" for (\d+) 分鐘 in "([^"]*)"$`, forIn)
	ctx.Step(`^"([^"]*)" 已登入並取得 Token "([^"]*)"$`, token)
	ctx.Step(`^已存在 (\d+)對(\d+) 聊天房間 with "([^"]*)" and "([^"]*)"$`, withAnd)
}
