import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    // 全局超时设置
    testTimeout: 120_000,
    
    // 优化速度
    maxThreads: 1,       // e2e测试需要顺序执行，避免端口冲突
    minThreads: 1,
    
    // 改善输出
    reporters: ['verbose'],
    
    // 改善诊断
    logHeapUsage: true,
    
    // 调试
    bail: 0,             // 允许测试继续执行，即使有失败
    hookTimeout: 60_000, // hooks超时时间增加
    teardownTimeout: 30_000,
    
    // 慢测试警告阈值
    slowTestThreshold: 10_000, // 10秒以上为慢测试
    
    // 细粒度控制
    environmentOptions: {
      teardown: 'always',
    },
    
    // 添加重试机制
    retry: 1,            // 失败测试自动重试一次
  },
}); 