import { memo, useEffect, useState } from "react";

interface CountdownDisplayProps {
	/** Whether auto-refresh is enabled */
	enabled: boolean;
	/** Refresh interval in milliseconds */
	interval: number;
	/** Whether user is currently interacting (pauses countdown) */
	paused?: boolean;
	/** CSS class for the countdown text */
	className?: string;
	/** Text to show when paused */
	pausedText?: string;
}

/**
 * Isolated countdown display component that manages its own timer state
 * to prevent re-rendering parent components every second.
 */
export const CountdownDisplay = memo(function CountdownDisplay({
	enabled,
	interval,
	paused = false,
	className = "text-info text-sm",
	pausedText = "Auto-refresh paused",
}: CountdownDisplayProps) {
	const [countdown, setCountdown] = useState(0);
	const [nextRefreshTime, setNextRefreshTime] = useState<Date | null>(null);

	// Set up the refresh timer
	useEffect(() => {
		if (!enabled || paused) {
			setNextRefreshTime(null);
			setCountdown(0);
			return;
		}

		// Set initial next refresh time
		setNextRefreshTime(new Date(Date.now() + interval));

		// Reset the timer every interval
		const resetTimer = setInterval(() => {
			setNextRefreshTime(new Date(Date.now() + interval));
		}, interval);

		return () => clearInterval(resetTimer);
	}, [enabled, interval, paused]);

	// Update countdown every second
	useEffect(() => {
		if (!nextRefreshTime || !enabled || paused) {
			setCountdown(0);
			return;
		}

		const updateCountdown = () => {
			const remaining = Math.max(0, Math.ceil((nextRefreshTime.getTime() - Date.now()) / 1000));
			setCountdown(remaining);

			// If countdown reaches 0, reset to the full interval
			if (remaining === 0) {
				setNextRefreshTime(new Date(Date.now() + interval));
			}
		};

		// Initial countdown update
		updateCountdown();
		const timer = setInterval(updateCountdown, 1000);

		return () => clearInterval(timer);
	}, [nextRefreshTime, enabled, paused, interval]);

	if (!enabled) {
		return null;
	}

	if (paused) {
		return <span className="ml-2 text-sm text-warning">• {pausedText}</span>;
	}

	if (countdown > 0) {
		return <span className={`ml-2 ${className}`}>• Auto-refresh in {countdown}s</span>;
	}

	return null;
});

CountdownDisplay.displayName = "CountdownDisplay";
