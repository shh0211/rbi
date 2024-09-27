<template>
  <n-layout>
    <n-layout-header class="header">
      <h3>script</h3>
      <n-button type="primary" @click="exportEditor">Export</n-button>
    </n-layout-header>
    <n-layout has-sider class="container">
      <n-layout-sider width="250px" class="column">
        <ul>
          <li
            v-for="n in listNodes"
            :key="n"
            draggable="true"
            :data-node="n.item"
            @dragstart="drag($event)"
            class="drag-drawflow"
          >
            <div class="node" :style="`background: ${n.color}`">{{ n.name }}</div>
          </li>
        </ul>
      </n-layout-sider>
      <n-layout>
        <div id="drawflow" @drop="drop($event)" @dragover="allowDrop($event)"></div>
      </n-layout>
    </n-layout>
  </n-layout>
  <n-modal :show="dialogVisible" title="Export" style="width: 50%">
    <n-card size="small">
      <span>Data:</span>
      <pre><code>{{dialogData}}</code></pre>
      <template #footer>
        <span class="dialog-footer">
          <n-button @click="dialogVisible = false">Cancel</n-button>
          <n-button type="primary" @click="dialogVisible = false">Confirm</n-button>
        </span>
      </template>
    </n-card>
  </n-modal>
</template>
<script>
  import Drawflow from 'drawflow';
  import { onMounted, shallowRef, h, getCurrentInstance, render, readonly, ref } from 'vue';
  import Node1 from './nodes/node1.vue';
  import Node2 from './nodes/node2.vue';
  import Node3 from './nodes/node3.vue';
  import 'drawflow/dist/drawflow.min.css';
  import './style.css';
  export default {
    name: 'DrawflowDashboard',
    setup() {
      const listNodes = readonly([
        {
          name: 'Get/Post',
          color: '#49494970',
          item: 'Node1',
          input: 0,
          output: 1,
        },
        {
          name: 'Script',
          color: 'blue',
          item: 'Node2',
          input: 1,
          output: 2,
        },
        {
          name: 'console.log',
          color: '#ff9900',
          item: 'Node3',
          input: 1,
          output: 0,
        },
      ]);
      const editor = shallowRef({});
      const dialogVisible = ref(false);
      const dialogData = ref({});
      const Vue = { version: 3, h, render };
      const internalInstance = getCurrentInstance();
      internalInstance.appContext.app._context.config.globalProperties.$df = editor;

      function exportEditor() {
        dialogData.value = editor.value.export();
        console.log(dialogVisible.value);
        dialogVisible.value = true;
      }

      const drag = (ev) => {
        if (ev.type === 'touchstart') {
          mobile_item_selec = ev.target.closest('.drag-drawflow').getAttribute('data-node');
        } else {
          ev.dataTransfer.setData('node', ev.target.getAttribute('data-node'));
        }
      };
      const drop = (ev) => {
        if (ev.type === 'touchend') {
          var parentdrawflow = document
            .elementFromPoint(
              mobile_last_move.touches[0].clientX,
              mobile_last_move.touches[0].clientY
            )
            .closest('#drawflow');
          if (parentdrawflow != null) {
            addNodeToDrawFlow(
              mobile_item_selec,
              mobile_last_move.touches[0].clientX,
              mobile_last_move.touches[0].clientY
            );
          }
          mobile_item_selec = '';
        } else {
          ev.preventDefault();
          var data = ev.dataTransfer.getData('node');
          addNodeToDrawFlow(data, ev.clientX, ev.clientY);
        }
      };
      const allowDrop = (ev) => {
        ev.preventDefault();
      };

      let mobile_item_selec = '';
      let mobile_last_move = null;
      function positionMobile(ev) {
        mobile_last_move = ev;
      }

      function addNodeToDrawFlow(name, pos_x, pos_y) {
        pos_x =
          pos_x *
            (editor.value.precanvas.clientWidth /
              (editor.value.precanvas.clientWidth * editor.value.zoom)) -
          editor.value.precanvas.getBoundingClientRect().x *
            (editor.value.precanvas.clientWidth /
              (editor.value.precanvas.clientWidth * editor.value.zoom));
        pos_y =
          pos_y *
            (editor.value.precanvas.clientHeight /
              (editor.value.precanvas.clientHeight * editor.value.zoom)) -
          editor.value.precanvas.getBoundingClientRect().y *
            (editor.value.precanvas.clientHeight /
              (editor.value.precanvas.clientHeight * editor.value.zoom));

        const nodeSelected = listNodes.find((ele) => ele.item == name);
        editor.value.addNode(
          name,
          nodeSelected.input,
          nodeSelected.output,
          pos_x,
          pos_y,
          name,
          {},
          name,
          'vue'
        );
      }

      onMounted(() => {
        var elements = document.getElementsByClassName('drag-drawflow');
        for (var i = 0; i < elements.length; i++) {
          elements[i].addEventListener('touchend', drop, false);
          elements[i].addEventListener('touchmove', positionMobile, false);
          elements[i].addEventListener('touchstart', drag, false);
        }

        const id = document.getElementById('drawflow');
        editor.value = new Drawflow(id, Vue, internalInstance.appContext.app._context);
        editor.value.start();

        editor.value.registerNode('Node1', Node1, {}, {});
        editor.value.registerNode('Node2', Node2, {}, {});
        editor.value.registerNode('Node3', Node3, {}, {});

        editor.value.import({
          drawflow: {
            Home: {
              data: {},
            },
          },
        });
      });

      return {
        exportEditor,
        listNodes,
        drag,
        drop,
        allowDrop,
        dialogVisible,
        dialogData,
      };
    },
  };
</script>
<style scoped>
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #494949;
  }
  .container {
    min-height: calc(100vh - 100px);
  }
  .column {
    border-right: 1px solid #494949;
  }
  .column ul {
    padding-inline-start: 0px;
    padding: 10px 10px;
  }
  .column li {
    background: transparent;
  }

  .node {
    border-radius: 8px;
    border: 2px solid #494949;
    display: block;
    height: 60px;
    line-height: 40px;
    padding: 10px;
    margin: 10px 0px;
    cursor: move;
  }
  #drawflow {
    width: 100%;
    height: 100%;
    text-align: initial;
    background: #2b2c30;
    background-size: 20px 20px;
    background-image: radial-gradient(#494949 1px, transparent 1px);
  }
</style>
