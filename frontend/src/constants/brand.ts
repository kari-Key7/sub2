/**
 * 品牌常量：所有对外可见的产品名/标识统一从这里引用，
 * 避免在组件、i18n、存储 key 中散落硬编码。
 */

/** 产品名（默认站点名，后台 site_name 为空时的回退值） */
export const BRAND_NAME = 'KimAI'

/** 浏览器标题后缀 */
export const BRAND_TAGLINE = 'AI API Gateway'

/** localStorage / IndexedDB / Web Locks 等本地存储 key 前缀 */
export const STORAGE_PREFIX = 'kimai'

/** 管理端 WebSocket 子协议名（需与后端 ops_ws_handler 保持一致） */
export const ADMIN_WS_PROTOCOL = 'kimai-admin'

/** 账号/代理导入导出文件的 data_type 标识（需与后端 account_data.go 保持一致） */
export const EXPORT_DATA_TYPE = 'kimai-data'

/** 环境变量名前缀，用于生成 CLI 配置示例 */
export const ENV_PREFIX = 'KIMAI'

/** CLI 配置示例中的 provider 标识（小写） */
export const PROVIDER_SLUG = 'kimai'
