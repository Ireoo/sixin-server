const io = require("socket.io-client");
const axios = require("axios");

const BASE_URL = "http://localhost:80"; // 请根据实际情况修改端口号
const SOCKET_URL = `${BASE_URL}/socket.io`;

// 测试 HTTP API
async function testHTTPAPI() {
  console.log("测试 HTTP API");

  try {
    // 测试 Ping
    const pingResponse = await axios.get(`${BASE_URL}/api/ping`);
    console.log("Ping 响应:", pingResponse.data);

    // 测试获取用户列表
    const usersResponse = await axios.get(`${BASE_URL}/api/users`);
    console.log("用户列表:", usersResponse.data);

    // 测试创建用户
    const newUser = { name: "测试用户", email: "test@example.com" };
    const createUserResponse = await axios.post(
      `${BASE_URL}/api/users`,
      newUser
    );
    console.log("创建用户响应:", createUserResponse.data);

    // 测试获取特定用户
    const userId = createUserResponse.data.id || 1; // 使用创建的用户ID或默认值1
    const getUserResponse = await axios.get(`${BASE_URL}/api/users/${userId}`);
    console.log("获取用户响应:", getUserResponse.data);

    // 测试更新用户
    const updateUser = { name: "更新的用户名" };
    const updateUserResponse = await axios.put(
      `${BASE_URL}/api/users/${userId}`,
      updateUser
    );
    console.log("更新用户响应:", updateUserResponse.data);

    // 测试删除用户
    const deleteUserResponse = await axios.delete(
      `${BASE_URL}/api/users/${userId}`
    );
    console.log("删除用户响应:", deleteUserResponse.data);
  } catch (error) {
    console.error("HTTP API 测试出错:", error.message);
  }
}

// 测试 WebSocket 功能
function testWebSocket() {
  console.log("测试 WebSocket");

  const socket = io(BASE_URL, { transports: ["polling"], debug: true });

  socket.on("connect_error", (error) => {
    console.log("连接错误:", error);
  });

  socket.on("connect_timeout", (timeout) => {
    console.log("连接超时:", timeout);
  });

  socket.on("connect", () => {
    console.log("WebSocket 已连接");

    // 测试自身信息
    socket.emit("self");

    // 测试接收设备状态
    socket.emit("receive");

    // 测试邮件通知状态
    socket.emit("email");

    // 测试发送消息
    const testMessage = {
      message: {
        msgId: "",
        talkerId: 1,
        listenerId: 2,
        roomId: 1,
        text: {
          message: "测试消息",
          image: "",
        },
        timestamp: Date.now(),
        type: 1,
        mentionIdList: "",
      },
    };
    socket.emit("message", JSON.stringify(testMessage));

    // 测试获取聊天记录
    socket.emit("getChats");

    // 测试获取房间列表
    socket.emit("getRooms");

    // 测试获取用户列表
    socket.emit("getUsers");
  });

  socket.on("self", (data) => {
    console.log("收到自身信息:", data);
  });

  socket.on("receive", (data) => {
    console.log("接收设备状态:", data);
  });

  socket.on("email", (data) => {
    console.log("邮件通知状态:", data);
  });

  socket.on("message", (data) => {
    console.log("收到消息:", data);
  });

  socket.on("getChats", (data) => {
    console.log("聊天记录:", data);
  });

  socket.on("getRooms", (data) => {
    console.log("房间列表:", data);
  });

  socket.on("getUsers", (data) => {
    console.log("用户列表:", data);
  });

  socket.on("error", (error) => {
    console.error("WebSocket 错误:", error);
  });

  socket.on("disconnect", () => {
    console.log("WebSocket 已断开连接");
  });
}

// 运行测试
async function runTests() {
  await testHTTPAPI();
  testWebSocket();
}

runTests();
