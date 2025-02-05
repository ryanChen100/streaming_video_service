Feature: 會員管理
  為了使用此串流影音平台
  作為一名使用者
  我希望能夠登入、登出，並正確管理我的使用者狀態

  Scenario: 成功登入
    Given 已存在一位擁有 email「user@example.com」與密碼「pass1234」的使用者
    When 我嘗試以「user@example.com」和「pass1234」進行登入
    Then 我應該得到「success」的回應
    And 我應該獲得一個有效的 session token

  Scenario: 錯誤密碼登入失敗
    Given 已存在一位擁有 email「user@example.com」與密碼「pass1234」的使用者
    When 我嘗試以「user@example.com」和「wrongpass」進行登入
    Then 我應該得到「failure」的回應
    And 我應該看到「invalid credentials」的錯誤訊息

  Scenario: 逾時登出
    Given 帳號「user@example.com」已成功登入
    And 會話（session）逾時設定為 30 分鐘
    When 超過 31 分鐘未進行任何操作
    Then 這個使用者的會話應視為無效
    And 使用者應被自動登出

  Scenario: 強制登出
    Given 帳號「user@example.com」已成功登入
    And 管理員（admin）下達了強制登出的指令
    Then 這個使用者的會話應被立即清除
    And 使用者在繼續操作前，必須重新登入

  Scenario: 斷線重連
    Given 帳號「user@example.com」已成功登入
    And 使用者暫時失去連線
    When 使用者在會話逾時前重新連線
    Then 使用者應仍然被視為處於登入狀態
    And 可以繼續使用同一個 session token