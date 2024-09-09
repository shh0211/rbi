import { RouteRecordRaw } from 'vue-router';
import { Layout } from '@/router/constant';
import { RecentlyViewed } from '@vicons/carbon';
import { renderIcon } from '@/utils/index';

const routes: Array<RouteRecordRaw> = [
  {
    path: '/recently',
    name: 'recently',
    component: Layout,
    meta: {
      title: '快速访问',
      icon: renderIcon(RecentlyViewed),
      sort: 0,
      is_root: true,
    },
    children: [{
      path: 'index',
      name: `recently`,
      meta: {
        title: '快速访问',
      },
      component: () => import('@/views/recently/index.vue'),
    }]
  },

];

export default routes;
