<template>
  <n-layout>
    <n-layout-header class="header">
      <h3>script</h3>
      <n-button type="primary" @click="saveEditor">Export</n-button>
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
  import Click from './nodes/click.vue';
  import Navigate from './nodes/navigate.vue';
  import SendKeys from './nodes/sendKeys.vue';
  import WaitVisible from './nodes/waitVisible.vue';
  import 'drawflow/dist/drawflow.min.css';
  import './style.css';
  import { updateScript } from '@/api/auto/automation';
  export default {
    name: 'DrawflowDashboard',
    props: {
      id: {
        type: String,
        required: true,
      },
    },
    setup() {
      const listNodes = readonly([
        {
          name: 'Navigate',
          color: '#ff9900',
          item: 'Navigate',
          input: 1,
          output: 1,
        },
        {
          name: 'Click',
          color: 'rgba(158,158,158,0.95)',
          item: 'Click',
          input: 1,
          output: 1,
        },
        {
          name: 'SendKeys',
          color: '#00ff33',
          item: 'SendKeys',
          input: 1,
          output: 1,
        },
        {
          name: 'WaitVisible',
          color: '#0095ff',
          item: 'WaitVisible',
          input: 1,
          output: 1,
        },
      ]);
      const editor = shallowRef({});
      const dialogVisible = ref(false);
      const dialogData = ref({});
      const Vue = { version: 3, h, render };
      const internalInstance = getCurrentInstance();
      internalInstance.appContext.app._context.config.globalProperties.$df = editor;

      function saveEditor() {
        dialogData.value = editor.value.export();
        console.log('数据', dialogData.value.drawflow.Home.data);
        updateScript(dialogData.value.drawflow.Home.data);
        dialogVisible.value = true;
      }
      // region
      function determineSequenceFromConnections(data) {
        // 首先创建一个字典来存储节点的入度
        const inDegree = {};
        const sequenceMap = {};

        // 初始化所有节点的入度为 0
        Object.keys(data).forEach((key) => {
          inDegree[key] = 0;
        });

        // 计算每个节点的入度
        Object.keys(data).forEach((key) => {
          const node = data[key];
          if (node.outputs) {
            Object.keys(node.outputs).forEach((outputKey) => {
              node.outputs[outputKey].connections.forEach((connection) => {
                inDegree[connection.node]++; // 增加目标节点的入度
              });
            });
          }
        });

        // 找到所有入度为 0 的节点作为起点
        const queue = [];
        Object.keys(inDegree).forEach((key) => {
          if (inDegree[key] === 0) {
            queue.push(key);
          }
        });

        let sequence = 1;
        // 进行拓扑排序来确定执行顺序
        while (queue.length > 0) {
          const currentNode = queue.shift();
          sequenceMap[currentNode] = sequence++;

          const node = data[currentNode];
          if (node.outputs) {
            Object.keys(node.outputs).forEach((outputKey) => {
              node.outputs[outputKey].connections.forEach((connection) => {
                const targetNode = connection.node;
                inDegree[targetNode]--;
                if (inDegree[targetNode] === 0) {
                  queue.push(targetNode);
                }
              });
            });
          }
        }

        return sequenceMap;
      }

      function prepareUpdateActionsRequest(data, automationID) {
        const sequenceMap = determineSequenceFromConnections(data);
        const actions = [];

        Object.keys(data).forEach((key) => {
          const node = data[key];
          const action = {
            ActionID: node.id || 0, // 假设 actionID 可以从 `id` 获取
            AutomationID: automationID, // 直接使用传入的 automationID
            Sequence: sequenceMap[key] || 0, // 根据连接关系推断出来的执行顺序
            ActionType: node.name || '', // 根据节点的 name 设置 ActionType
            Selector: node.data?.method || '', // 假设 CSS 选择器保存在 data 中的 method 字段
            Value: node.html || '', // 使用 html 字段作为 Value
            URL: '', // 假设没有明确的 URL 字段，可以视情况进行调整
          };

          actions.push(action);
        });

        return {
          automation_id: automationID,
          actions: actions,
        };
      }

      // 示例用法
      const data = {
        1: {
          id: 1,
          name: 'Navigate',
          data: {},
          class: 'Navigate',
          html: 'Navigate',
          typenode: 'vue',
          inputs: {
            input_1: {
              connections: [],
            },
          },
          outputs: {
            output_1: {
              connections: [
                {
                  node: '2',
                  output: 'input_1',
                },
              ],
            },
          },
          pos_x: 221,
          pos_y: 175,
        },
        2: {
          id: 2,
          name: 'SendKeys',
          data: {
            method: 'ByQuery',
          },
          class: 'SendKeys',
          html: 'SendKeys',
          typenode: 'vue',
          inputs: {
            input_1: {
              connections: [
                {
                  node: '1',
                  input: 'output_1',
                },
              ],
            },
          },
          outputs: {
            output_1: {
              connections: [],
            },
          },
          pos_x: 731,
          pos_y: 182,
        },
      };

      const automationID = 123; // 替换为实际的 automationID
      const updateActionsRequest = prepareUpdateActionsRequest(data, automationID);

      console.log(updateActionsRequest);

      // endregion
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
        editor.value.registerNode('Navigate', Navigate, {}, {});
        editor.value.registerNode('Click', Click, {}, {});
        editor.value.registerNode('SendKeys', SendKeys, {}, {});
        editor.value.registerNode('WaitVisible', WaitVisible, {}, {});
        editor.value.import({
          drawflow: {
            Home: {
              data: {},
            },
          },
        });
      });

      return {
        saveEditor,
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
    width: 5000px;
    height: 550px;
    text-align: initial;
    background: #2b2c30;
    background-size: 20px 20px;
    background-image: radial-gradient(#494949 1px, transparent 1px);
  }
</style>
