import { ElMessage } from 'element-plus'
import { sha3_256 } from 'js-sha3';


export class SimpleMutex {
    private locked = false;

    // 尝试获取锁，成功返回 true，失败返回 false
    tryLock(): boolean {
        if (this.locked) return false;
        this.locked = true;
        return true;
    }

    // 释放锁
    unlock(): void {
        this.locked = false;
    }
}


export class DualMutex {
    private locked = false;
    private waitQueue: (() => void)[] = [];

    /**
     * 尝试获取锁（非阻塞）
     * 成功返回 true，失败返回 false
     */
    tryLock(): boolean {
        if (this.locked) return false;
        this.locked = true;
        return true;
    }

    /**
     * 获取锁（阻塞等待）
     * 一直等待直到获取锁
     */
    async waitLock(): Promise<void> {
        if (!this.locked) {
            this.locked = true;
            return;
        }

        // 加入等待队列
        return new Promise<void>(resolve => {
            this.waitQueue.push(() => {
                this.locked = true;  // 获取锁
                resolve();
            });
        });
    }

    /**
     * 释放锁
     * 如果有等待的，唤醒下一个
     */
    unlock(): void {
        if (this.waitQueue.length > 0) {
            // 有等待的，直接唤醒下一个
            const next = this.waitQueue.shift()!;
            next();  // 这里会设置 locked = true
        } else {
            // 没有等待的，释放锁
            this.locked = false;
        }
    }
}


export function sha3_256_sync(User_Id: number, Passwd: string): string {
    return sha3_256(User_Id.toString() + "." + Passwd)
}


export function General_status_code_processing(error: unknown): boolean {
    const axiosError = error as {
        response?: {
            data?: { Msg?: string },
            status?: number,


        },
        code?: string
    };

    switch (axiosError.code) {
        case "ERR_NETWORK":
            ElMessage({ message: '后端未运行', type: 'error' })
            return false
    }
    ElMessage({
        message: axiosError.response?.data?.Msg || '请求失败',
        type: 'warning'
    })

    return true

}












