import api from '../api';

export function getData() {
  return api.get('/list');
}

export function launchContainer() {
  return api.post(
    `/start?fileUrl=${encodeURIComponent(
      'https://pub-a0628cecf1764cf3936ade50c81a9a8e.r2.dev/5.%E4%BA%91%E6%A1%8C%E9%9D%A2%E7%B3%BB%E7%BB%9F%E4%BD%BF%E7%94%A8%E6%89%8B%E5%86%8C.docx'
    )}`
  );
}

export function stopContainer(containerId: string) {
  return api.post('/stop', { containerId });
}
