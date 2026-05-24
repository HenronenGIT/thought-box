type MicrophoneIconProps = {
  className?: string;
};

export function MicrophoneIcon({ className }: MicrophoneIconProps) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M12 14.7a4.1 4.1 0 0 0 4.1-4.1V6.1a4.1 4.1 0 0 0-8.2 0v4.5a4.1 4.1 0 0 0 4.1 4.1Z"
        fill="currentColor"
      />
      <path
        d="M5.7 10.5a1 1 0 0 1 2 0 4.3 4.3 0 0 0 8.6 0 1 1 0 1 1 2 0 6.3 6.3 0 0 1-5.3 6.2V20a1 1 0 1 1-2 0v-3.3a6.3 6.3 0 0 1-5.3-6.2Z"
        fill="currentColor"
      />
    </svg>
  );
}
