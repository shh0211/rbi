import api from '../api';

export function getIsInit() {
  return api.get('/user/check');
}
