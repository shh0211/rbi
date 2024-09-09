import { RouteRecordRaw } from 'vue-router';
import { Layout } from '@/router/constant';
import { Carbon3DCurveAutoColon } from '@vicons/carbon';
import { renderIcon } from '@/utils/index';

const routes: Array<RouteRecordRaw> = [
  {
    path: '/auto',
    name: 'auto',
    component: Layout,
    meta: {
      title: '自动配置',
      icon: renderIcon(Carbon3DCurveAutoColon),
      sort: 4,
      is_root: true,
    },
    children: [{
      path: 'index',
      name: `auto`,
      meta: {
        title: '自动化',
      },
      component: () => import('@/views/auto/index.vue'),
    }]
  },

];

export default routes;
