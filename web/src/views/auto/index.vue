<script lang="ts">
  import { defineComponent, h, onMounted, ref } from 'vue';
  import { NButton, useMessage } from 'naive-ui';
  import type { DataTableColumns } from 'naive-ui';
  import { toRaw } from 'vue';
  import { addNewScript, getScripts, delScript } from '@/api/auto/automation';
  import { ArrowBack } from '@vicons/ionicons5';
  import DrawflowDashboard from '@/views/auto/components/drawflow.vue';
  interface Automation {
    AutomationID: string;
    Name: string;
    Description: string;
    CreatedAt: string;
  }

  function createColumns(
    edit: (row: Automation) => void,
    del: (row: Automation) => void
  ): DataTableColumns<Automation> {
    return [
      {
        title: '#',
        key: 'AutomationID',
        resizable: true,
      },
      {
        title: '脚本名称',
        key: 'Name',
        resizable: true,
      },
      {
        title: '描述',
        key: 'Description',
        resizable: true,
      },
      {
        title: '创建时间',
        key: 'CreatedAt',
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
                onClick: () => edit(row),
              },
              { default: () => '编辑' }
            ),
            h(
              NButton,
              {
                size: 'small',
                type: 'error',
                style: { marginLeft: '8px' },
                onClick: () => del(row),
              },
              { default: () => '删除' }
            ),
          ];
        },
      },
    ];
  }

  const data = ref<Automation[]>([]);
  const name = ref('');
  const description = ref('');
  const isEdit = ref(false);
  const editId = ref('');
  export default defineComponent({
    components: {
      DrawflowDashboard,
      ArrowBack,
    },
    setup() {
      const showModal = ref(false);
      const message = useMessage();

      function edit(row: Automation) {
        console.log('edit', row);
        editId.value = row.AutomationID;
        isEdit.value = true;
      }
      onMounted(() => {
        console.log('onMounted');
        getScripts().then((res) => {
          data.value = res.data;
        });
      });
      function del(row: Automation) {
        row = toRaw(row);
        console.log(row);
        delScript(row.AutomationID)
          .then(() => {
            message.success('已删除');
            refresh();
          })
          .catch((error) => {
            message.error('删除失败', error);
          });
      }
      function changeDialog() {
        showModal.value = !showModal.value;
      }
      function newScript() {
        console.log('newScript', name.value, description.value);
        const automation = {
          Name: name.value,
          Description: description.value,
          CreatedAt: new Date().toISOString(), // Capture current date and time
        };
        console.log('automation', automation);
        addNewScript(automation)
          .then(() => {
            message.success('新建成功');
            refresh();
          })
          .catch((error) => {
            message.error('新建失败', error);
          });
        showModal.value = !showModal.value;
      }
      function refresh() {
        getScripts().then((res) => {
          data.value = res.data;
        });
      }
      function changeShow() {
        isEdit.value = !isEdit.value;
      }
      return {
        data,
        newScript,
        changeDialog,
        showModal,
        refresh,
        name,
        description,
        isEdit,
        editId,
        changeShow,
        columns: createColumns(edit, del),
        pagination: false as const,
      };
    },
  });
</script>

<template>
  <div v-show="!isEdit">
    <n-page-header>
      <n-button round type="info" style="font-size: medium" @click="refresh">刷新</n-button>
      <n-button
        round
        type="warning"
        style="margin-top: 10px; margin-left: 10px; font-size: medium"
        @click="changeDialog"
        >新建</n-button
      >
      <n-divider />
    </n-page-header>
    <n-modal v-model:show="showModal">
      <n-card
        style="width: 600px"
        title="新建脚本"
        :bordered="false"
        size="huge"
        role="dialog"
        aria-modal="true"
      >
        <n-input v-model:value="name" round style="margin-bottom: 10px" placeholder="名称" />
        <n-input v-model:value="description" round placeholder="描述" />
        <template #footer>
          <div style="margin-left: 35%">
            <n-button round type="primary" style="margin-top: 10px" @click="newScript"
              >确定</n-button
            >
            <n-button round style="margin-top: 10px; margin-left: 10px" @click="changeDialog"
              >取消</n-button
            >
          </div>
        </template>
      </n-card>
    </n-modal>
    <n-data-table :columns="columns" :data="data" :pagination="pagination" :bordered="false" />
  </div>
  <div v-show="isEdit">
    <n-page-header>
      <n-button text style="font-size: 24px" @click="changeShow">
        <n-icon>
          <ArrowBack />
        </n-icon>
      </n-button>
    </n-page-header>
    <drawflow-dashboard :id="editId" />
  </div>
</template>

<style scoped lang="less"></style>
