type BrandLogoProps = {
  size?: number
}

export default function BrandLogo({ size = 20 }: BrandLogoProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <rect x="1.5" y="1.5" width="21" height="21" rx="7" fill="url(#rgperp-bg)" />
      <path
        d="M7 17V7H11.8C14.15 7 15.6 8.22 15.6 10.28C15.6 11.84 14.7 12.88 13.21 13.18L16.5 17H13.84L10.84 13.45H9.3V17H7ZM9.3 11.48H11.57C12.84 11.48 13.45 11.08 13.45 10.23C13.45 9.39 12.84 8.98 11.57 8.98H9.3V11.48Z"
        fill="white"
      />
      <path d="M17.4 7.4L19.9 9.9L15.25 14.55L13.95 13.25L17.4 9.8L16.1 8.5L17.4 7.4Z" fill="#7CFFF1" />
      <defs>
        <linearGradient id="rgperp-bg" x1="3" y1="3" x2="21" y2="21" gradientUnits="userSpaceOnUse">
          <stop stopColor="#20C9B5" />
          <stop offset="0.52" stopColor="#1A7F89" />
          <stop offset="1" stopColor="#103548" />
        </linearGradient>
      </defs>
    </svg>
  )
}
