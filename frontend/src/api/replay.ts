import client from './client';
import { CANSignal } from '../types';

export const getFrame = (sceneId: number, frameIndex: number) =>
  client.get<Blob>(`/replay/scene/${sceneId}/frame/${frameIndex}`, {
    responseType: 'blob',
  });

export const getCANSignals = (sceneId: number) =>
  client.get<CANSignal[]>(`/replay/scene/${sceneId}/can`);

export const getReplayData = (sceneId: number) =>
  client.get<{
    frame_count: number;
    duration: number;
    fps: number;
  }>(`/replay/scene/${sceneId}/info`);
