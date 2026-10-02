import { api } from './client';
import type { DeliveryPlan, DeliveryPlanResponse } from '$lib/types/delivery-plan';
const path = (slug: string, id: string) =>
	`/api/workspaces/${encodeURIComponent(slug)}/projects/${encodeURIComponent(id)}/delivery-plan`;
export async function getDeliveryPlan(slug: string, id: string): Promise<DeliveryPlanResponse> {
	const response = await api.get<DeliveryPlanResponse>(path(slug, id));
	return response;
}
export async function saveDeliveryPlan(
	slug: string,
	id: string,
	plan: DeliveryPlan,
	version: number
): Promise<DeliveryPlanResponse> {
	const response = await api.patch<DeliveryPlanResponse>(path(slug, id), { plan, version });
	return response;
}
