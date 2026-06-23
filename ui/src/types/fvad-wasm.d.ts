declare module '@echogarden/fvad-wasm' {
  export interface FvadModule {
    // 内存访问
    HEAP8: Int8Array;
    HEAP16: Int16Array;
    HEAPU8: Uint8Array;
    HEAPU16: Uint16Array;
    HEAP32: Int32Array;
    HEAPU32: Uint32Array;
    HEAPF32: Float32Array;
    HEAPF64: Float64Array;

    // VAD 实例管理
    _fvad_new(): number;
    _fvad_free(inst: number): void;
    _fvad_reset(inst: number): void;

    // VAD 配置
    _fvad_set_mode(inst: number, mode: number): number;
    _fvad_set_sample_rate(inst: number, sampleRate: number): number;

    // VAD 处理
    _fvad_process(inst: number, framePtr: number, frameLen: number): number;

    // 内存管理
    _malloc(size: number): number;
    _free(ptr: number): void;

    // 工具函数
    setValue(ptr: number, value: number | number[], type: string): void;
    getValue(ptr: number, type: string): number;
  }

  export default function fvad(): Promise<FvadModule>;
}