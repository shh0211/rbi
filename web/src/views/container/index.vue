<script lang="ts">
  import { defineComponent, h, onMounted, ref } from 'vue';
  import { NButton, useMessage } from 'naive-ui';
  import type { DataTableColumns } from 'naive-ui';
  import { getData, launchContainer, stopContainer } from '@/api/container/container';

  interface Container {
    ID: string;
    ContainerId: string;
    ExpireAt: string;
  }
  const loadingMap = ref({});
  function createColumns(
    run: (row: Container) => void,
    stop: (row: Container) => void
  ): DataTableColumns<Container> {
    return [
      {
        title: '#',
        key: 'ID',
        resizable: true,
      },
      {
        title: '容器ID',
        key: 'ContainerId',
        resizable: true,
      },
      {
        title: '超期时间',
        key: 'ExpireAt',
        resizable: true,
      },
      {
        title: 'Action',
        key: 'actions',
        render(row) {
          return [
            h(
              NButton,
              {
                size: 'small',
                type: 'primary',
                onClick: () => run(row),
              },
              { default: () => '启动' }
            ),
            h(
              NButton,
              {
                size: 'small',
                type: 'error',
                style: { marginLeft: '8px' },
                onClick: () => stop(row),
                loading: loadingMap.value[row.ContainerId]
                  ? loadingMap.value[row.ContainerId]
                  : false,
              },
              { default: () => '删除' }
            ),
          ];
        },
      },
    ];
  }

  const data = ref<Container[]>([]);
  const launchLoading = ref(false);
  export default defineComponent({
    setup() {
      const message = useMessage();
      function run(row: Container) {
        message.info('启动' + row.ID + '容器');
        const url = import.meta.env.VITE_API_BASE_URL + '/' + row.ContainerId + '/';
        console.log('url:', url);
        const newWindow = window.open(url, '_blank');
        if (newWindow) {
          newWindow.focus(); // 确保新窗口获得焦点
        } else {
          console.log('Failed to open the window');
        }
      }
      onMounted(() => {
        console.log('onMounted');
        getData().then((res) => {
          data.value = res.data;
        });
      });
      function stop(row: Container) {
        loadingMap.value[row.ContainerId] = true;
        message.info('删除' + row.ID + '容器');
        stopContainer(row.ContainerId)
          .then(() => {
            refresh();
            loadingMap.value[row.ContainerId] = false;
          })
          .catch(() => {
            message.error('容器删除失败');
            loadingMap.value[row.ContainerId] = false;
          });
      }
      function launch() {
        launchLoading.value = true;
        launchContainer()
          .then(() => {
            refresh();
            launchLoading.value = false;
          })
          .catch(() => {
            message.error('容器启动失败');
            launchLoading.value = false;
          });
      }
      function refresh() {
        getData().then((res) => {
          console.log('res', res);
          data.value = res.data;
          console.log('data:', data);
        });
      }
      return {
        data,
        launch,
        refresh,
        launchLoading,
        columns: createColumns(run, stop),
        pagination: false as const,
      };
    },
  });
</script>

<template>
  <n-page-header>
    <n-button round type="info" style="font-size: medium" @click="refresh">刷新</n-button>
    <n-button
      round
      style="margin-top: 10px; margin-left: 10px; font-size: medium"
      type="warning"
      @click="launch"
      :loading="launchLoading"
      >启动容器</n-button
    >
    <n-divider />
  </n-page-header>
  <n-data-table :columns="columns" :data="data" :pagination="pagination" :bordered="false" />
</template>

<style scoped lang="less"></style>
