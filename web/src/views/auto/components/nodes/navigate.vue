<template>
  <div ref="el">
    <nodeHeader title="Navigate" />
    <n-input v-model:value="target" @update:value="updateTargetInput" placeholder="目标网址" size="small" />
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
      const target = ref('');
      const method = ref('get');
      const dataNode = ref({});
      df = getCurrentInstance().appContext.config.globalProperties.$df.value;

      const updateTargetInput = (value) => {
        dataNode.value.data.target = value;
        df.updateNodeDataFromId(nodeId.value, dataNode.value);
      };

      const updateSelect = (value) => {
        dataNode.value.data.method = value;
        df.updateNodeDataFromId(nodeId.value, dataNode.value);
      };

      onMounted(async () => {
        await nextTick();
        nodeId.value = el.value.parentElement.parentElement.id.slice(5);
        dataNode.value = df.getNodeFromId(nodeId.value);

        target.value = dataNode.value.data.target;
        method.value = dataNode.value.data.method;
      });

      return {
        el,
        target,
        method,
        updateSelect,
        updateTargetInput,
      };
    },
  });
</script>
