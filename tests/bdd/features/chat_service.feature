Feature: 聊天功能
  In order to communicate effectively
  As registered users and group admins
  I want to start private conversations and manage group chats

  Background:
    Given "memberA" 已登入並取得 Token "tokenA"
    And "memberB" 已登入並取得 Token "tokenB"
    And "adminUser" 已登入並取得 Token "adminToken"
    And "normalUser" 已登入並取得 Token "userToken"
    And a 群組聊天室 "Go Club" 已存在 with "adminUser" as Admin and "normalUser" as Member

  # 1 對 1 聊天場景
  Scenario: 成功建立 1對1 聊天
    When "memberA" 建立 1對1 聊天邀請 "memberB"
    Then 聊天房間應該包含 "memberA" 和 "memberB"

  Scenario: 發送與接收訊息
    Given 已存在 1對1 聊天房間 with "memberA" and "memberB"
    When "memberA" 發送訊息 "Hello B!"
    Then "memberB" 應該收到訊息 "Hello B!"

  # 群組聊天場景
  Scenario: Admin 禁言普通會員
    When "adminUser" 禁言 "normalUser" for 10 分鐘 in "Go Club"
    Then "normalUser" 無法在 "Go Club" 發送訊息