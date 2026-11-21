import { defineStore } from "pinia";
import { type userType, store, router, resetRouter, routerArrays, storageLocal } from "../utils";
import type { UserResult, RefreshTokenResult } from "@/api/user";
import { authApi } from "@/api/config";
import { useMultiTagsStoreHook } from "./multiTags";
import { type DataInfo, setToken, removeToken, getToken, userKey } from "@/utils/auth";

export const useUserStore = defineStore("pure-user", {
  state: (): userType => ({
    // 头像
    avatar: storageLocal().getItem<DataInfo<number>>(userKey)?.avatar ?? "",
    // 用户名
    username: storageLocal().getItem<DataInfo<number>>(userKey)?.username ?? "",
    // 昵称
    nickname: storageLocal().getItem<DataInfo<number>>(userKey)?.nickname ?? "",
    // 页面级别权限
    roles: storageLocal().getItem<DataInfo<number>>(userKey)?.roles ?? [],
    // 按钮级别权限
    permissions: storageLocal().getItem<DataInfo<number>>(userKey)?.permissions ?? [],
    // 前端生成的验证码（按实际需求替换）
    verifyCode: "",
    // 判断登录页面显示哪个组件（0：登录（默认）、1：手机登录、2：二维码登录、3：注册、4：忘记密码）
    currentPage: 0,
    // 是否勾选了登录页的免登录
    isRemembered: false,
    // 登录页的免登录存储几天，默认7天
    loginDay: 7
  }),
  actions: {
    /** 存储头像 */
    SET_AVATAR(avatar: string) {
      this.avatar = avatar;
    },
    /** 存储用户名 */
    SET_USERNAME(username: string) {
      this.username = username;
    },
    /** 存储昵称 */
    SET_NICKNAME(nickname: string) {
      this.nickname = nickname;
    },
    /** 存储角色 */
    SET_ROLES(roles: Array<string>) {
      this.roles = roles;
    },
    /** 存储按钮级别权限 */
    SET_PERMS(permissions: Array<string>) {
      this.permissions = permissions;
    },
    /** 存储前端生成的验证码 */
    SET_VERIFYCODE(verifyCode: string) {
      this.verifyCode = verifyCode;
    },
    /** 存储登录页面显示哪个组件 */
    SET_CURRENTPAGE(value: number) {
      this.currentPage = value;
    },
    /** 存储是否勾选了登录页的免登录 */
    SET_ISREMEMBERED(bool: boolean) {
      this.isRemembered = bool;
    },
    /** 设置登录页的免登录存储几天 */
    SET_LOGINDAY(value: number) {
      this.loginDay = Number(value);
    },
    /** 登入：调用采集服务 auth/login（access_key=用户名） */
    async loginByUsername(data) {
      return new Promise<UserResult>((resolve, reject) => {
        authApi
          .login({ access_key: data.username, password: data.password })
          .then((res: any) => {
            const d = res?.Data || {};
            const payload = {
              accessToken: d.access_token,
              refreshToken: d.refresh_token,
              expires: new Date(Date.now() + Number(d.expires_in || 900) * 1000),
              avatar: "",
              username: d.access_key,
              nickname: d.access_key,
              // RBAC：直接采用后端登录返回的角色/权限（后端已按“关闭或超管回退通配”处理）。
              // roles 驱动菜单可见性(meta.roles 交集)，permissions 驱动按钮级 v-perms。
              roles: Array.isArray(d.roles) ? d.roles : [],
              permissions: Array.isArray(d.permissions) ? d.permissions : []
            };
            setToken(payload);
            resolve({ code: 0, message: "ok", data: payload } as unknown as UserResult);
          })
          .catch(error => reject(error));
      });
    },
    /** 登出：先通知后端使令牌失效（尽力而为），再清理本地状态 */
    async logOut() {
      try {
        // access_token 由 http 拦截器自动放入 Authorization 头，此处仅传 refresh_token
        await authApi.logout({ refresh_token: getToken()?.refreshToken });
      } catch {
        // 后端不可用或令牌已过期不阻断本地登出
      }
      this.username = "";
      this.roles = [];
      this.permissions = [];
      removeToken();
      useMultiTagsStoreHook().handleTags("equal", [...routerArrays]);
      await router.push("/login");
      resetRouter();
    },
    /** 刷新`token`：调用采集服务 auth/refresh-token */
    async handRefreshToken(data) {
      return new Promise<RefreshTokenResult>((resolve, reject) => {
        authApi
          .refresh({ refresh_token: data.refreshToken })
          .then((res: any) => {
            const d = res?.Data || {};
            const expires = new Date(Date.now() + Number(d.expires_in || 900) * 1000);
            setToken({
              accessToken: d.access_token,
              refreshToken: d.refresh_token,
              expires
            });
            resolve({
              code: 0,
              message: "ok",
              data: {
                accessToken: d.access_token,
                refreshToken: d.refresh_token,
                expires
              }
            } as unknown as RefreshTokenResult);
          })
          .catch(error => reject(error));
      });
    }
  }
});

export function useUserStoreHook() {
  return useUserStore(store);
}
