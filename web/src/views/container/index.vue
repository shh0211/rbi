<script lang="ts">
import { defineComponent, h } from 'vue'
import { NButton, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'

interface Container {
  no: number
  application: string
  status: string
  startTime: string
}

function createColumns(run: (row: Container) => void, stop: (row: Container) => void): DataTableColumns<Container> {
  return [
    {
      title: '#',
      key: 'no',
      resizable: true
    },
    {
      title: '应用',
      key: 'application',
      resizable: true
    },
    {
      title: '状态',
      key: 'status',
      resizable: true
    },
    {
      title: '启动时间',
      key: 'startTime',
      resizable: true
    },
    {
      title: 'Action',
      key: 'actions',
      render(row) {
        return [h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            onClick: () => run(row)
          },
          { default: () => '启动' },
        ),
        h(
          NButton,
          {
            size: 'small',
            type: 'error',
            style: { marginLeft: '8px' },
            onClick: () => stop(row)
          },
          { default: () => '删除' },
        )]
      }
    }
  ]
}

const data: Container[] = [
  {
    no: 1,
    application: 'app1',
    status: 'running',
    startTime: '2021-09-01 12:00:00'
  }
]

export default defineComponent({
  setup() {
    const message = useMessage()
    function run(row:Container) {
      message.info('启动'+row.no+"容器")
    }

    function stop(row:Container) {
      message.info('删除'+row.no+"容器")
    }
    return {
      data,
      columns: createColumns(run,stop),
      pagination: false as const
    }
  }
})
</script>

<template>
  <n-data-table
    :columns="columns"
    :data="data"
    :pagination="pagination"
    :bordered="false"
  />
</template>

<style scoped lang="less">

</style>
