<template>
  <div ref="el">
    <nodeHeader title="Get/Post" />
    <n-select
      v-model:value="method"
      placeholder="Select"
      @update-value="updateSelect"
      size="small"
      :options="options"
      df-method
    />
    <br /><br />
    <n-input v-model:value="url" df-url placeholder="Please input" size="small">
      <template #prefix>https://</template>
    </n-input>
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
      const url = ref('');
      const method = ref('get');
      const dataNode = ref({});
      const options = readonly([
        {
          value: 'get',
          label: 'GET',
        },
        {
          value: 'post',
          label: 'POST',
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

        url.value = dataNode.value.data.url;
        method.value = dataNode.value.data.method;
      });

      return {
        el,
        url,
        method,
        options,
        updateSelect,
      };
    },
  });
</script>
