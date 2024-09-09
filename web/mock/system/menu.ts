import { resultSuccess } from '../_util';

const menuList = () => {
  const result: any[] = [];

  return result;
};

export default [
  {
    url: '/api/menu/list',
    timeout: 1000,
    method: 'get',
    response: () => {
      const list = menuList();
      return resultSuccess({
        list,
      });
    },
  },
];
