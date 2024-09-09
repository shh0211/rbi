import { RouteRecordRaw } from 'vue-router';
import { Layout } from '@/router/constant';
import { AuditOutlined } from '@vicons/antd';
import { renderIcon } from '@/utils/index';

const routes: Array<RouteRecordRaw> = [
  {
    path: '/audit',
    name: 'audit',
    component: Layout,
    meta: {
      title: '审计服务',
      icon: renderIcon(AuditOutlined),
      sort: 6,
      is_root: true
    },
    children: [{
      path: 'index',
      name: `audit`,
      meta: {
        title: '审计服务',
      },
      component: () => import('@/views/container/index.vue'),
    }]
  },

];

export default routes;
