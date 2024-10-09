import api from '../api';

export function getGraph(id: string) {
  return api.get(`/graph/get?automation_id=${id}`);
}

export function updateGraph(data: any) {
  return api.post('/graph/update', data);
}
