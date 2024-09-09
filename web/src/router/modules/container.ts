import { RouteRecordRaw } from 'vue-router';
import { Layout } from '@/router/constant';
import { ContainerRegistry } from '@vicons/carbon';
import { renderIcon } from '@/utils/index';

const routes: Array<RouteRecordRaw> = [
  {
    path: '/container',
    name: 'container',
    component: Layout,
    meta: {
      title: '容器管理',
      icon: renderIcon(ContainerRegistry),
      sort: 1,
      is_root: true
    },
    children: [{
      path: 'index',
      name: `container`,
      meta: {
        title: '容器管理',
      },
      component: () => import('@/views/container/index.vue'),
    }]
  },

];

export default routes;
