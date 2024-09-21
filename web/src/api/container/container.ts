import api from '../api';

export function getData() {
  return api.get('/list');
}

export function launchContainer() {
  return api.post(
    `/start?fileUrl=${encodeURIComponent('https://www.gov.cn/zhengce/pdfFile/2022_PDF.pdf')}`
  );
}

export function stopContainer(containerId: string) {
  return api.post('/stop', { containerId });
}
