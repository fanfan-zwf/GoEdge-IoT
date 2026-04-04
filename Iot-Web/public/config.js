
var ip = '192.168.50.185'
var port = 8101

// 这个文件打包后会在根目录，可直接修改
window.APP_CONFIG = {
  // 默认地址
  http_Front_url: `http://${ip}:${port}`,
  
  // 配置服务地址
  config_service_url: `http://${ip}:${port}`,
}