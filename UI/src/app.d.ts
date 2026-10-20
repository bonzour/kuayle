// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		interface PageState {
			timesheet?: import('$lib/types/timesheets').TimesheetHistoryState;
		}
		// interface Platform {}
	}
}

export {};
