import request from '@/utils/request'
import type { DictionaryItem } from '@/types'

interface DictFilter {
  dictType?: string
  dictGroup?: string
  status?: number
  page?: number
  pageSize?: number
}

export const dictApi = {
  listTypes: () => request.get('/system/dictionary/types'),
  listByType: (dictType: string) =>
    request.get(`/system/dictionary/data/${dictType}`),
  list: (params?: DictFilter) => request.get('/system/dictionary', { params }),
  create: (data: Partial<DictionaryItem>) =>
    request.post('/system/dictionary', data),
  update: (dicId: number, data: Partial<DictionaryItem>) =>
    request.put(`/system/dictionary/${dicId}`, data),
  delete: (dicId: number) => request.delete(`/system/dictionary/${dicId}`),
}
