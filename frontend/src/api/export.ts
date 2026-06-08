import client from './client';
import { ExportTask } from '../types';

export const exportScene = (sceneId: number, format: 'openscenario' | 'rosbag') =>
  client.post<ExportTask>(`/export/scene/${sceneId}`, { format });

export const getExportStatus = (taskId: number) =>
  client.get<ExportTask>(`/export/task/${taskId}`);

export const downloadExport = (taskId: number) =>
  client.get<Blob>(`/export/task/${taskId}/download`, {
    responseType: 'blob',
  });
