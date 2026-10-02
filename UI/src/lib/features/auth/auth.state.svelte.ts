import type { User } from '$lib/types/auth';
import { getMe } from '$lib/api/auth';

class AuthState {
	user = $state<User | null>(null);
	loading = $state(true);
	initError = $state(false);
	authenticated = $derived(this.user !== null);

	async init() {
		this.loading = true;
		this.initError = false;
		try {
			this.user = await getMe();
		} catch (error) {
			if (error instanceof Error && error.message === 'Unauthorized') this.user = null;
			else this.initError = true;
		} finally {
			this.loading = false;
		}
	}

	setUser(user: User | null) {
		this.user = user;
	}

	clear() {
		this.user = null;
	}
}

export const authState = new AuthState();
