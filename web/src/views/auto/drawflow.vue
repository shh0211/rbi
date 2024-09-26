<template>
  <div class="h-full w-full">
    <div class="flex justify-end mb-3 text-lg text-gray-100">
      <input
        class="text-sm mr-2 rounded-sm text-gray-700 hover:bg-gray-100"
        placeholder="Add program name"
        @input="addProgramName($event)"
        v-model="nodeProgramName"
      />
      <button
        class="w-32 bg-green-500 mr-3 rounded-md hover:bg-green-400 cursor-pointer"
        @click="saveProgramName"
      >
        Save
      </button>
      <select
        class="w-32 bg-blue-400 mr-3 rounded-md hover:bg-blue-300 cursor-pointer"
        @change="valueSelected($event)"
      >
        <option value="Select" class="text-center">Choose</option>
        <option v-for="program in programOptions" :key="program.id" :value="program.id">
          {{ `${program.programName}#${program.uid}` }}
        </option>
      </select>
      <button class="w-32 bg-red-400 mr-3 rounded-md hover:bg-red-300" @click="deleteProgram"
        >Delete</button
      >
    </div>

    <div class="h-3/4 flex flex-row w-full">
      <div class="w-[200px] mx-auto p-2 text-sm">
        <div
          class="nodes-list"
          draggable="true"
          v-for="node in nodesList"
          :key="node.item"
          :node-item="node.item"
          @dragstart="drag($event)"
        >
          <span class="node">{{ node.name }}</span>
        </div>
      </div>
      <div class="drawflow-container w-full mx-2 relative">
        <div id="drawflow" @drop="drop($event)" @dragover="allowDrop($event)"></div>
        <button
          class="absolute w-20 bg-blue-400 m-2 rounded-md text-white text-sm right-0 top-0 hover:bg-blue-300"
          @click="cleanEditor"
        >
          Clear
        </button>
      </div>
    </div>
  </div>
</template>

<script>
  import { h, render, getCurrentInstance, onMounted, ref } from 'vue';
  import Drawflow from 'drawflow';
  import NodeNumber from './Node-number.vue';
  import NodeOperation from './Node-operation.vue';
  import NodeAssign from './Node-assign.vue';
  import NodeIf from './Node-if.vue';
  import NodeCondition from './Node-condition.vue';
  import NodeFor from './Node-for.vue';

  export default {
    name: 'DrawflowDashboard',
    setup() {
      const editor = ref(null);
      const nodeProgramName = ref('');
      const programOptions = ref([]);
      const optionSelected = ref(0);

      const internalInstance = getCurrentInstance();
      const Vue = { version: 3, h, render };
      internalInstance.appContext.app._context.config.globalProperties.$df = editor;

      const drag = (ev) => {
        ev.dataTransfer.setData('node', ev.target.getAttribute('node-item'));
      };

      const allowDrop = (ev) => {
        ev.preventDefault();
      };

      const drop = (ev) => {
        ev.preventDefault();
        const data = ev.dataTransfer.getData('node');
        addNodeToDrawFlow(data, ev.clientX, ev.clientY);
      };

      const addNodeToDrawFlow = (name, pos_x, pos_y) => {
        const nodeSelected = nodesList.find((object) => object.item === name);
        editor.value.addNode(
          name,
          nodeSelected.input,
          nodeSelected.output,
          pos_x,
          pos_y,
          name,
          { number: 0, num1: 0, num2: 0 },
          name,
          'vue'
        );
      };

      const valueSelected = (event) => {
        optionSelected.value = event.target.value;
        showSelected();
      };

      const addProgramName = (event) => {
        nodeProgramName.value = event.target.value;
      };

      const saveProgramName = () => {
        if (!nodeProgramName.value) {
          alert('Please enter a program name.');
          return;
        }
        setData();
        nodeProgramName.value = '';
      };

      const setData = async () => {
        await fetch('http://localhost:5000/setAllPrograms', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(editor.value.export()),
        });
      };

      const showSelected = () => {
        // Replace with logic to fetch and display selected program
      };

      const cleanEditor = () => {
        editor.value.clear();
      };

      const deleteProgram = () => {
        deleteData();
        cleanEditor();
        getData();
      };

      onMounted(() => {
        editor.value = new Drawflow(document.getElementById('drawflow'), Vue);
        editor.value.start();
        editor.value.registerNode('number', NodeNumber);
        editor.value.registerNode('addition', NodeOperation, {}, { title: 'Addition' });
        editor.value.registerNode('subtraction', NodeOperation, {}, { title: 'Subtraction' });
        editor.value.registerNode('multiplication', NodeOperation, {}, { title: 'Multiplication' });
        editor.value.registerNode('division', NodeOperation, {}, { title: 'Division' });
        editor.value.registerNode('assign', NodeAssign);
        editor.value.registerNode('if', NodeIf, {}, { title: 'If statement' });
        editor.value.registerNode('for', NodeFor, {}, { title: 'For statement' });
        editor.value.registerNode('nodeCondition', NodeCondition);
      });

      return {
        nodeProgramName,
        nodesList,
        drag,
        drop,
        allowDrop,
        addProgramName,
        saveProgramName,
        valueSelected,
        cleanEditor,
        deleteProgram,
        programOptions,
      };
    },
  };
</script>

<style scoped>
  .node {
    background-color: #4a8ac2;
    color: #f7f7f7;
    padding: 5px;
    border-radius: 8px;
    border: 2px solid #4b769bc4;
    display: block;
    height: 50px;
    margin: 10px 0px;
    cursor: move;
  }
  @media only screen and (min-width: 350px) {
    .node {
      font-size: small;
    }
    .drawflow-container {
      height: 500px;
    }
  }

  @media only screen and (min-width: 600px) {
    .node {
      font-size: medium;
    }
    .drawflow-container {
      height: 700px;
    }
  }

  .node:hover {
    background-color: #649cce;
  }

  #drawflow {
    text-align: initial;
    width: 100%;
    height: 100%;
    background: #f1eeee;
    background-size: 20px 20px;
    background-image: radial-gradient(#c5c3c3 1px, transparent 1px);
  }
</style>
