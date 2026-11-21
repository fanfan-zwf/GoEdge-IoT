import { createI18n } from 'vue-i18n'
import zh from './zh'
import en from './en'
import fr from './fr'
import ru from './ru'
import es from './es'
import ar from './ar'

// 从 localStorage 读取用户语言偏好，默认英语
const savedLocale = localStorage.getItem('locale') || 'en'

const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'en',
  messages: {
    zh,
    en,
    fr,
    ru,
    es,
    ar
  }
})

export default i18n

// 切换语言工具函数
export function setLocale(locale: 'zh' | 'en' | 'fr' | 'ru' | 'es' | 'ar') {
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
}
