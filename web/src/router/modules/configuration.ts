import { RouteRecordRaw } from 'vue-router';
import { Layout } from '@/router/constant';
import { AppstoreOutlined } from '@vicons/antd';
import { renderIcon } from '@/utils/index';

const routes: Array<RouteRecordRaw> = [
  {
    path: '/configuration',
    name: 'configuration',
    component: Layout,
    meta: {
      title: '配置管理',
      icon: renderIcon(AppstoreOutlined),
      sort: 3,
      is_root: true
    },
    children: [{
      path: 'index',
      name: `configuration`,
      meta: {
        title: '配置管理',
      },
      component: () => import('@/views/configuration/index.vue'),
    }]
  },

];

export default routes;
