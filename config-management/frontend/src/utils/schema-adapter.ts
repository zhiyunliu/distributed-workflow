import type {
  DynamicFormField,
  DynamicFormSchema,
  DynamicTableColumn,
  LegacyFormField,
  PersistedFormSchema,
  SchemaObject,
  SchemaValue,
} from '@/types/form'

type NullableSchemaValue = SchemaValue | null | undefined

export interface SchemaCarrier {
  schema?: NullableSchemaValue
  formSchema?: NullableSchemaValue
  formName?: string
}

const LEGACY_TO_DYNAMIC_TYPE: Record<LegacyFormField['type'], DynamicFormField['type']> = {
  text: 'input',
  textarea: 'textarea',
  number: 'input',
  select: 'select',
  date: 'date',
  checkbox: 'selectMultiple',
  radio: 'select',
}

const DYNAMIC_TO_LEGACY_TYPE: Record<DynamicFormField['type'], LegacyFormField['type']> = {
  input: 'text',
  textarea: 'textarea',
  date: 'date',
  select: 'select',
  switch: 'checkbox',
  selectMultiple: 'checkbox',
  table: 'textarea',
}

function isObject(value: unknown): value is SchemaObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function parseSchemaValue(value: NullableSchemaValue): SchemaObject {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return {}
    }
    try {
      const parsed: unknown = JSON.parse(trimmed)
      return isObject(parsed) ? parsed : {}
    } catch {
      return {}
    }
  }

  return isObject(value) ? value : {}
}

function pickEffectiveRawSchema(input: SchemaCarrier | NullableSchemaValue): SchemaObject {
  if (typeof input === 'string' || isObject(input) || input == null) {
    return parseSchemaValue(input)
  }

  const hasSchema = input.schema !== undefined && input.schema !== null
  const source = hasSchema ? input.schema : input.formSchema
  return parseSchemaValue(source)
}

function asString(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback
}

function asBoolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function asOptions(value: unknown): Array<{ label: string; value: string }> {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item) => {
      if (!isObject(item)) {
        return null
      }
      return {
        label: asString(item.label),
        value: asString(item.value),
      }
    })
    .filter((item): item is { label: string; value: string } => item !== null)
}

function normalizeColumns(value: unknown): DynamicTableColumn[] | undefined {
  if (!Array.isArray(value)) {
    return undefined
  }

  const columns: DynamicTableColumn[] = []
  for (const column of value) {
    if (!isObject(column)) {
      continue
    }

    columns.push({
      key: asString(column.key),
      label: asString(column.label),
      type: asString(column.type) === 'select' ? 'select' : 'input',
      placeholder: asString(column.placeholder) || undefined,
    })
  }

  return columns
}

function normalizeDynamicField(field: unknown, index: number): DynamicFormField {
  const source = isObject(field) ? field : {}
  const key = asString(source.key) || asString(source.name) || `field_${index}`
  const rawType = asString(source.type)

  const dynamicType: DynamicFormField['type'] =
    rawType in DYNAMIC_TO_LEGACY_TYPE
      ? (rawType as DynamicFormField['type'])
      : rawType in LEGACY_TO_DYNAMIC_TYPE
        ? LEGACY_TO_DYNAMIC_TYPE[rawType as LegacyFormField['type']]
        : 'input'

  return {
    key,
    label: asString(source.label, key),
    type: dynamicType,
    required: asBoolean(source.required),
    placeholder: asString(source.placeholder),
    defaultValue: source.defaultValue,
    apiParams: isObject(source.apiParams)
      ? (source.apiParams as Record<string, string | number | boolean>)
      : undefined,
    regex: asString(source.regex),
    regexMsg: asString(source.regexMsg),
    columns: normalizeColumns(source.columns),
    options: asOptions(source.options),
  }
}

function normalizeLegacyField(field: unknown, index: number): LegacyFormField {
  const source = isObject(field) ? field : {}
  const name = asString(source.name) || asString(source.key) || `field_${index}`
  const rawType = asString(source.type)

  const legacyType: LegacyFormField['type'] =
    rawType in LEGACY_TO_DYNAMIC_TYPE
      ? (rawType as LegacyFormField['type'])
      : rawType in DYNAMIC_TO_LEGACY_TYPE
        ? DYNAMIC_TO_LEGACY_TYPE[rawType as DynamicFormField['type']]
        : 'text'

  return {
    name,
    label: asString(source.label, name),
    type: legacyType,
    required: asBoolean(source.required),
    options: asOptions(source.options),
    placeholder: asString(source.placeholder),
  }
}

export function normalizeToDynamicSchema(input: SchemaCarrier | NullableSchemaValue): DynamicFormSchema {
  const source = pickEffectiveRawSchema(input)
  const fields = Array.isArray(source.fields)
    ? source.fields.map((field, index) => normalizeDynamicField(field, index))
    : []

  const formName =
    typeof input === 'object' && input !== null && 'formName' in input
      ? asString((input as SchemaCarrier).formName) || asString(source.formName)
      : asString(source.formName)

  return {
    formName,
    commonApi: asString(source.commonApi),
    fields,
  }
}

export function normalizeToPersistedSchema(input: SchemaCarrier | NullableSchemaValue): PersistedFormSchema {
  const source = pickEffectiveRawSchema(input)
  const fields = Array.isArray(source.fields)
    ? source.fields.map((field, index) => normalizeLegacyField(field, index))
    : []

  const formName =
    typeof input === 'object' && input !== null && 'formName' in input
      ? asString((input as SchemaCarrier).formName) || asString(source.formName)
      : asString(source.formName)

  return {
    formName,
    commonApi: asString(source.commonApi),
    fields,
  }
}

export function stringifyPersistedSchema(input: SchemaCarrier | NullableSchemaValue): string {
  return JSON.stringify(normalizeToPersistedSchema(input))
}
