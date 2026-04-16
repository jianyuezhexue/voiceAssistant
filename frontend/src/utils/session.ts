/**
 * 生成并管理客户端会话 ID
 * 格式: deviceId_ipAddress
 *
 * 说明：浏览器安全限制下无法获取真实 MAC 地址，
 * 使用 localStorage 持久化的 deviceId 作为设备唯一标识替代。
 */

const DEVICE_ID_KEY = 'va_device_id';

/**
 * 生成伪 MAC 地址格式的设备 ID (12 位十六进制)
 */
function generateDeviceId(): string {
  const hex = '0123456789ABCDEF';
  let id = '';
  for (let i = 0; i < 12; i++) {
    id += hex.charAt(Math.floor(Math.random() * 16));
  }
  // 格式化为 xx:xx:xx:xx:xx:xx
  return id.match(/.{2}/g)!.join(':');
}

/**
 * 获取设备唯一标识（模拟 MAC 地址）
 */
export function getDeviceId(): string {
  let deviceId = localStorage.getItem(DEVICE_ID_KEY);
  if (!deviceId) {
    deviceId = generateDeviceId();
    localStorage.setItem(DEVICE_ID_KEY, deviceId);
  }
  return deviceId;
}


/**
 * 生成 sessionId: deviceId
 */
export async function getSessionId(): Promise<string> {
  return getDeviceId()
}
