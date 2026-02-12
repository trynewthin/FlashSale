declare module "json-bigint" {
  export interface JSONBigParseOptions {
    storeAsString?: boolean
  }

  export interface JSONBigInstance {
    parse(text: string): unknown
    stringify(value: unknown): string
  }

  export default function JSONbig(options?: JSONBigParseOptions): JSONBigInstance
}
