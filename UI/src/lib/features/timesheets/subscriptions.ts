// Each socket is scoped by its authorized workspace URL. Domain payloads never
// enter the timesheet: the server channel only asks it to reread authorized HTTP.
export function createTimesheetSubscriptions(
	oninvalidate: () => void,
	onstatus: (connected: boolean) => void,
	connect = (slug: string): WebSocket => {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		return new WebSocket(`${protocol}//${window.location.host}/api/workspaces/${encodeURIComponent(slug)}/ws?channel=timesheets`);
	}
) {
	type Connection = { socket?: WebSocket; timer?: ReturnType<typeof setTimeout>; connected: boolean };
	const connections = new Map<string, Connection>();
	let user: string | undefined;
	const status = () => onstatus([...connections.values()].every(connection => connection.connected));

	function remove(slug: string, connection: Connection) {
		connections.delete(slug);
		clearTimeout(connection.timer);
		connection.socket?.close();
	}

	function open(slug: string, connection: Connection) {
		if (connections.get(slug) !== connection) return;
		const socket = connect(slug);
		connection.socket = socket;
		const current = () => connections.get(slug) === connection && connection.socket === socket;
		socket.onopen = () => {
			if (!current()) return;
			connection.connected = true;
			status();
			oninvalidate(); // Covers the gap between HTTP read and subscription.
		};
		socket.onmessage = event => {
			if (!current()) return;
			try {
				const message = JSON.parse(event.data);
				if (message.type === 'app.refresh' && message.payload?.resources?.includes('timesheets')) oninvalidate();
			} catch { /* Ignore malformed frames. */ }
		};
		socket.onerror = () => { if (current()) socket.close(); };
		socket.onclose = () => {
			if (!current()) return;
			connection.connected = false;
			status();
			oninvalidate(); // A proxy may hide a policy close; always recheck access.
			connection.timer = setTimeout(() => open(slug, connection), 3000);
		};
	}

	return {
		update(userId: string | undefined, slugs: string[]) {
			if (user !== userId) {
				for (const [slug, connection] of connections) remove(slug, connection);
				user = userId;
			}
			const selected = new Set(userId ? slugs : []);
			for (const [slug, connection] of connections) if (!selected.has(slug)) remove(slug, connection);
			for (const slug of selected) {
				if (connections.has(slug)) continue;
				const connection: Connection = { connected: false };
				connections.set(slug, connection);
				open(slug, connection);
			}
			status();
		},
		close() {
			for (const [slug, connection] of connections) remove(slug, connection);
		}
	};
}
