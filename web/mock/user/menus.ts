import { resultSuccess } from '../_util';

const menusList = [
];

export default [
  {
    url: '/api/menus',
    timeout: 1000,
    method: 'get',
    response: () => {
      return resultSuccess(menusList);
    },
  },
];
