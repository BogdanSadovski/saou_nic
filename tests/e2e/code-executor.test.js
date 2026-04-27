#!/usr/bin/env node

/**
 * E2E Test: Code Executor Integration
 * 
 * Flow:
 * 1. Register user
 * 2. Create interview session
 * 3. Submit code (Python snippet)
 * 4. Verify execution result
 * 5. Check database persistence
 * 6. Test JavaScript execution
 * 7. Verify multiple submissions
 */

const API_BASE = process.env.API_BASE || 'http://localhost:8000/api';
const EXEC_BASE = process.env.EXEC_BASE || 'http://localhost:8083';

class E2ETest {
  constructor() {
    this.token = null;
    this.sessionId = null;
    this.userId = null;
    this.testsPassed = 0;
    this.testsFailed = 0;
  }

  async assert(condition, message) {
    if (condition) {
      console.log(`✅ ${message}`);
      this.testsPassed++;
    } else {
      console.error(`❌ ${message}`);
      this.testsFailed++;
      throw new Error(message);
    }
  }

  async request(method, path, body = null) {
    const headers = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const options = {
      method,
      headers,
    };

    if (body) {
      options.body = JSON.stringify(body);
    }

    const response = await fetch(`${API_BASE}${path}`, options);
    const data = await response.json();

    return { status: response.status, data };
  }

  async executeCode(code, language) {
    const response = await fetch(`${EXEC_BASE}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        language,
        code,
        input: '',
      }),
    });

    const data = await response.json();
    return data;
  }

  async testRegister() {
    console.log('\n📝 Test 1: Register User');
    const email = `e2e_${Date.now()}@test.com`;
    const password = 'Test@Pass123';
    const username = `testuser_${Date.now()}`;

    const { status, data } = await this.request('POST', '/auth/register', {
      email,
      password,
      username,
    });

    await this.assert(status === 201 || status === 200, `Register returned ${status}`);
    await this.assert(data.access_token || data.accessToken, 'Got JWT token');

    this.token = data.access_token || data.accessToken;
    this.userId = data.user_id || data.userId;

    console.log(`  Token: ${this.token.substring(0, 20)}...`);
  }

  async testCreateSession() {
    console.log('\n🎬 Test 2: Create Interview Session');

    const { status, data } = await this.request(
      'POST',
      '/interviews/sessions',
      {
        role: 'Backend Engineer',
        level: 'Middle',
        duration_minutes: 15,
        question_limit: 4,
      }
    );

    await this.assert(status === 201 || status === 200, `Session created with status ${status}`);
    await this.assert(data.session_id || data.data?.session_id, 'Got session ID');

    this.sessionId = data.session_id || data.data?.session_id;
    console.log(`  Session ID: ${this.sessionId}`);
  }

  async testPythonExecution() {
    console.log('\n🐍 Test 3: Execute Python Code');

    const pythonCode = `
result = 2 + 2
print(f"Result: {result}")
for i in range(3):
    print(f"Iteration {i}")
`.trim();

    const result = await this.executeCode(pythonCode, 'python');

    await this.assert(result.status === 'success', `Python execution status: ${result.status}`);
    await this.assert(result.output.includes('Result: 4'), 'Output contains expected result');
    await this.assert(result.exit_code === 0, `Exit code is 0`);
    await this.assert(result.runtime > 0, `Runtime measured: ${result.runtime}ms`);

    console.log(`  Output: ${result.output.substring(0, 50)}...`);
    console.log(`  Runtime: ${result.runtime}ms`);
  }

  async testJavaScriptExecution() {
    console.log('\n🟨 Test 4: Execute JavaScript Code');

    const jsCode = `
const sum = (a, b) => a + b;
console.log('Sum:', sum(5, 3));
console.log('Array test:', [1,2,3].map(x => x * 2));
`.trim();

    const result = await this.executeCode(jsCode, 'javascript');

    await this.assert(result.status === 'success', `JavaScript execution status: ${result.status}`);
    await this.assert(result.output.includes('Sum: 8'), 'Output contains correct sum');
    await this.assert(result.exit_code === 0, `Exit code is 0`);

    console.log(`  Output: ${result.output.substring(0, 50)}...`);
  }

  async testCodeSubmission() {
    console.log('\n📤 Test 5: Submit Code to Interview Session');

    const { status, data } = await this.request(
      'POST',
      `/interviews/sessions/${this.sessionId}/submit-code`,
      {
        language: 'python',
        code: 'print("Hello from interview!")',
        input: '',
      }
    );

    await this.assert(
      status === 200 || status === 201,
      `Code submission returned ${status}`
    );
    await this.assert(data.submission_id, 'Got submission ID');
    await this.assert(data.status === 'success', `Execution status: ${data.status}`);
    await this.assert(
      data.output.includes('Hello from interview'),
      'Output contains code result'
    );

    console.log(`  Submission ID: ${data.submission_id}`);
    console.log(`  Output: ${data.output}`);
  }

  async testGetSubmissions() {
    console.log('\n📋 Test 6: Get Code Submissions History');

    const { status, data } = await this.request(
      'GET',
      `/interviews/sessions/${this.sessionId}/code-submissions`
    );

    await this.assert(status === 200, `Get submissions returned ${status}`);
    await this.assert(Array.isArray(data.submissions), 'Got submissions array');
    await this.assert(data.count >= 1, `At least 1 submission recorded (count: ${data.count})`);

    console.log(`  Total submissions: ${data.count}`);
  }

  async testErrorHandling() {
    console.log('\n⚠️ Test 7: Error Handling');

    // Test disallowed pattern
    const { status: status1, data: data1 } = await this.request(
      'POST',
      `/interviews/sessions/${this.sessionId}/submit-code`,
      {
        language: 'python',
        code: 'import os; os.system("echo hacked")',
      }
    );

    await this.assert(
      status1 === 400 || data1.error,
      `Disallowed pattern detected (status: ${status1})`
    );

    // Test timeout
    const slowCode = 'import time; time.sleep(15)';
    const result = await this.executeCode(slowCode, 'python');
    
    await this.assert(
      result.status === 'timeout' || result.exit_code === 124,
      `Timeout detected (status: ${result.status})`
    );

    console.log(`  Security checks working ✓`);
  }

  async testCodeExecutorHealth() {
    console.log('\n💓 Test 8: Code Executor Health Check');

    try {
      const response = await fetch(`${EXEC_BASE}/healthz`);
      await this.assert(response.status === 200, `Code executor is healthy`);
    } catch (err) {
      await this.assert(false, `Code executor health check failed: ${err.message}`);
    }
  }

  async run() {
    console.log('🚀 Starting E2E Code Executor Tests\n');
    console.log(`API Base: ${API_BASE}`);
    console.log(`Executor Base: ${EXEC_BASE}\n`);

    try {
      // Wait for services to be ready
      console.log('⏳ Waiting for services to be ready...');
      await this.sleep(2000);

      await this.testCodeExecutorHealth();
      await this.testRegister();
      await this.testCreateSession();
      await this.testPythonExecution();
      await this.testJavaScriptExecution();
      await this.testCodeSubmission();
      await this.testGetSubmissions();
      await this.testErrorHandling();

      console.log(`\n${'='.repeat(50)}`);
      console.log(`📊 Test Results: ${this.testsPassed}/${this.testsPassed + this.testsFailed} passed`);
      console.log(`${'='.repeat(50)}\n`);

      if (this.testsFailed === 0) {
        console.log('✨ All tests passed! Code executor is working correctly.\n');
        process.exit(0);
      } else {
        console.error(`❌ ${this.testsFailed} test(s) failed\n`);
        process.exit(1);
      }
    } catch (error) {
      console.error('\n💥 Test suite failed:', error.message);
      console.error(error.stack);
      process.exit(1);
    }
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

const test = new E2ETest();
test.run();
