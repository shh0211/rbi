<template>
  <div ref="el">
    <nodeHeader title="SendKey" />
    <n-input v-model:value="selector" placeholder="目标元素选择器" size="small" />
    <br /><br />
    <n-input v-model:value="key" placeholder="发送键值" size="small" />
    <br /><br />
    <n-select
      v-model:value="method"
      placeholder="查询方式"
      @update-value="updateSelect"
      size="small"
      :options="options"
      df-method
    />
  </div>
</template>

<script>
  import { defineComponent, onMounted, getCurrentInstance, readonly, ref, nextTick } from 'vue';
  import nodeHeader from './nodeHeader.vue';

  export default defineComponent({
    components: {
      nodeHeader,
    },
    setup() {
      const el = ref(null);
      const nodeId = ref(0);
      let df = null;
      const selector = ref('');
      const key = ref('');
      const method = ref('get');
      const dataNode = ref({});
      const options = readonly([
        {
          value: 'ByQuery',
          label: 'ByQuery',
        },
        {
          value: 'ByID',
          label: 'ByID',
        },
        {
          label: 'BySearch',
          value: 'BySearch',
        },
        {
          label: 'ByJSPath',
          value: 'ByJSPath',
        },
        {
          label: 'ByXPath',
          value: 'ByXPath',
        },
      ]);

      df = getCurrentInstance().appContext.config.globalProperties.$df.value;

      const updateSelect = (value) => {
        dataNode.value.data.method = value;
        df.updateNodeDataFromId(nodeId.value, dataNode.value);
      };

      onMounted(async () => {
        await nextTick();
        nodeId.value = el.value.parentElement.parentElement.id.slice(5);
        dataNode.value = df.getNodeFromId(nodeId.value);

        selector.value = dataNode.value.data.selector;
        key.value = dataNode.value.data.key;
        method.value = dataNode.value.data.method;
      });

      return {
        el,
        selector,
        key,
        method,
        options,
        updateSelect,
      };
    },
  });
</script>
