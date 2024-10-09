import api from '../api';

export function getScripts() {
  return api.get('/automation/getScripts');
}

export function addNewScript(data: any) {
  return api.post('/automation/newScript', data);
}

export function delScript(ID: string) {
  return api.post(`/automation/delScript?id=${ID}`);
}

export function updateScript(data: any) {
  return api.post('/automation/updateScript', data);
}
