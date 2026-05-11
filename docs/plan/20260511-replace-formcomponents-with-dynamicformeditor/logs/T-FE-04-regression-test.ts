// 文件位置建议: config-management/frontend/src/components/__tests__/DynamicFormEditor.regression.spec.ts
// 当前仓库尚未配置 vitest/jest，本文件作为回归测试用例草案落盘，待测试框架接入后直接迁移执行。

import { mount } from '@vue/test-utils'
import DynamicFormEditor from '../DynamicFormEditor.vue'

describe('DynamicFormEditor - 展示层回归', () => {
  it('should render all field types correctly', () => {
    const schema = {
      formName: 'Test Form',
      fields: [
        { key: 'name', type: 'input', label: 'Name', defaultValue: '' },
        { key: 'desc', type: 'textarea', label: 'Description', defaultValue: '' },
        { key: 'date', type: 'date', label: 'Date', defaultValue: '' },
        { key: 'status', type: 'select', label: 'Status', defaultValue: '', options: [{ label: 'Active', value: 'active' }] },
        { key: 'tags', type: 'selectMultiple', label: 'Tags', defaultValue: [], options: [{ label: 'A', value: 'a' }] },
        { key: 'active', type: 'switch', label: 'Active', defaultValue: false },
        { key: 'items', type: 'table', label: 'Items', columns: [{ key: 'item', label: 'Item', type: 'input' }], defaultValue: [] },
      ],
    }

    const editingData = {
      name: 'Test',
      desc: 'Desc',
      date: '2024-01-01',
      status: 'active',
      tags: ['a'],
      active: true,
      items: [{ item: 'item1' }],
    }

    const wrapper = mount(DynamicFormEditor, {
      props: { schema, editingData, readonly: true },
    })

    expect(wrapper.find('h3').text()).toBe('Test Form')
    expect(wrapper.findAll('input').length).toBeGreaterThan(0)
    expect(wrapper.findAll('table').length).toBeGreaterThan(0)
  })

  it('should enforce readonly mode by disabling all inputs', () => {
    const schema = { fields: [{ key: 'name', type: 'input', label: 'Name', defaultValue: '' }] }
    const editingData = { name: 'Test' }

    const wrapper = mount(DynamicFormEditor, {
      props: { schema, editingData, readonly: true },
    })

    const inputs = wrapper.findAll('input')
    inputs.forEach((input) => {
      expect(input.attributes('disabled')).toBeDefined()
    })
  })

  it('should not have add/delete/submit buttons', () => {
    const schema = {
      fields: [
        {
          key: 'items',
          type: 'table',
          label: 'Items',
          columns: [{ key: 'item', label: 'Item', type: 'input' }],
          defaultValue: [],
        },
      ],
    }
    const editingData = { items: [] }

    const wrapper = mount(DynamicFormEditor, {
      props: { schema, editingData, readonly: true },
    })

    expect(wrapper.find('[data-testid="add-table-row"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="delete-table-row"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="submit-form"]').exists()).toBe(false)
  })
})
