import client from './client';
import { Annotation } from '../types';

export const createAnnotation = (data: Partial<Annotation>) =>
  client.post<Annotation>('/annotations', data);

export const updateAnnotation = (id: number, data: Partial<Annotation>) =>
  client.put<Annotation>(`/annotations/${id}`, data);

export const getAnnotationsByScene = (sceneId: number) =>
  client.get<Annotation[]>(`/annotations/scene/${sceneId}`);

export const deleteAnnotation = (id: number) =>
  client.delete(`/annotations/${id}`);
