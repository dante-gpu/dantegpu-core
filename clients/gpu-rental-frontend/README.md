# DanteGPU Frontend

## Overview

DanteGPU Frontend is a modern, responsive web application built with React and TypeScript that provides a comprehensive interface for GPU rental services. The platform enables users to browse, rent, and manage GPU resources for high-performance computing, machine learning, and AI workloads.

## Table of Contents

- [Features](#features)
- [Technology Stack](#technology-stack)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Development](#development)
- [Build and Deployment](#build-and-deployment)
- [Project Structure](#project-structure)
- [Configuration](#configuration)
- [API Integration](#api-integration)
- [Styling and Design](#styling-and-design)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Features

### Core Functionality
- **GPU Marketplace**: Browse and filter available GPU resources
- **Rental Management**: Real-time monitoring and control of active rentals
- **User Authentication**: Secure login and registration system
- **Dashboard Analytics**: Comprehensive usage statistics and cost tracking
- **Profile Management**: User account settings and billing information
- **Responsive Design**: Optimized for desktop, tablet, and mobile devices

### Advanced Features
- **Real-time Updates**: Live status monitoring of GPU instances
- **Cost Calculator**: Dynamic pricing estimation for rental duration
- **Resource Filtering**: Advanced search and filtering capabilities
- **Usage Analytics**: Detailed performance metrics and historical data
- **Notification System**: Toast notifications for important events
- **Dark/Light Theme**: Customizable user interface themes

## Technology Stack

### Frontend Framework
- **React 18.2.0**: Modern React with hooks and concurrent features
- **TypeScript 5.2.2**: Type-safe JavaScript development
- **Vite 4.5.14**: Fast build tool and development server

### UI and Styling
- **Tailwind CSS 3.4.1**: Utility-first CSS framework
- **Headless UI 1.7.18**: Unstyled, accessible UI components
- **Heroicons 2.0.18**: Beautiful hand-crafted SVG icons
- **Radix UI**: Low-level UI primitives for complex components

### State Management and Data Fetching
- **TanStack Query 4.36.1**: Powerful data synchronization for React
- **React Router DOM 6.20.1**: Declarative routing for React applications
- **React Context**: Built-in state management for authentication

### Development Tools
- **ESLint**: Code linting and quality enforcement
- **PostCSS**: CSS processing and optimization
- **Autoprefixer**: Automatic vendor prefix handling

## Prerequisites

Before running this project, ensure you have the following installed:

- **Node.js**: Version 18.0.0 or higher
- **npm**: Version 8.0.0 or higher (comes with Node.js)
- **Git**: For version control

### System Requirements
- **Operating System**: Windows 10+, macOS 10.15+, or Linux
- **Memory**: Minimum 4GB RAM (8GB recommended)
- **Storage**: At least 1GB free space for dependencies

## Installation

### 1. Clone the Repository
```bash
git clone https://github.com/your-organization/dantegpu-core.git
cd dantegpu-core/gpu-rental-frontend
```

### 2. Install Dependencies
```bash
npm install
```

### 3. Environment Configuration
Create a `.env` file in the root directory:
```bash
cp .env.example .env
```

Configure the following environment variables:
```env
VITE_API_BASE_URL=http://localhost:8000
VITE_APP_NAME=DanteGPU
VITE_APP_VERSION=1.0.0
VITE_ENABLE_ANALYTICS=false
```

## Development

### Start Development Server
```bash
npm run dev
```

The application will be available at `http://localhost:3001`

### Development Features
- **Hot Module Replacement**: Instant updates without page refresh
- **TypeScript Checking**: Real-time type checking and error reporting
- **ESLint Integration**: Automatic code quality checks
- **Source Maps**: Enhanced debugging experience

### Available Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start development server |
| `npm run build` | Build for production |
| `npm run preview` | Preview production build |
| `npm run lint` | Run ESLint checks |
| `npm run type-check` | Run TypeScript checks |

## Build and Deployment

### Production Build
```bash
npm run build
```

The build artifacts will be stored in the `dist/` directory.

### Docker Deployment
```bash
# Build Docker image
docker build -t dantegpu-frontend .

# Run container
docker run -p 80:80 dantegpu-frontend
```

### Environment-Specific Builds
```bash
# Staging environment
npm run build:staging

# Production environment
npm run build:production
```

## Project Structure

```
gpu-rental-frontend/
├── public/                 # Static assets
│   └── vite.svg           # Favicon and icons
├── src/
│   ├── components/        # Reusable UI components
│   │   ├── ui/           # Base UI components
│   │   ├── Layout.tsx    # Main layout component
│   │   └── Navbar.tsx    # Navigation component
│   ├── contexts/         # React context providers
│   │   └── AuthContext.tsx
│   ├── pages/            # Page components
│   │   ├── Dashboard.tsx
│   │   ├── GPUMarketplace.tsx
│   │   ├── Login.tsx
│   │   └── Register.tsx
│   ├── lib/              # Utility functions
│   │   └── utils.ts
│   ├── font/             # Custom fonts
│   │   └── AeonikTRIAL-Light.otf
│   ├── App.tsx           # Main application component
│   ├── main.tsx          # Application entry point
│   └── index.css         # Global styles
├── index.html            # HTML template
├── package.json          # Dependencies and scripts
├── tailwind.config.js    # Tailwind CSS configuration
├── tsconfig.json         # TypeScript configuration
├── vite.config.ts        # Vite configuration
└── README.md             # Project documentation
```

## Configuration

### Tailwind CSS Configuration

The project uses a custom Tailwind configuration with a cream-based color palette:

```javascript
// tailwind.config.js
module.exports = {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        'sans': ['Aeonik', 'sans-serif'],
        'aeonik': ['Aeonik', 'sans-serif'],
      },
      colors: {
        cream: {
          50: '#faf8f5',
          100: '#f5f2ed',
          200: '#ede7dd',
          300: '#e0d6c7',
          400: '#d1c2ad',
          500: '#c2ad93',
          600: '#8b7355',
          700: '#6b5b47',
          800: '#4a3f35',
          900: '#2d2620',
        },
      },
    },
  },
  plugins: [],
}
```

### TypeScript Configuration

The project uses strict TypeScript configuration for enhanced type safety:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  }
}
```

### Vite Configuration

Custom Vite configuration for optimal development and build performance:

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3001,
    host: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          router: ['react-router-dom'],
          ui: ['@headlessui/react', '@heroicons/react'],
        },
      },
    },
  },
})
```

## API Integration

### Authentication Context

The application uses React Context for authentication state management:

```typescript
interface AuthContextType {
  user: User | null
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name: string) => Promise<void>
  logout: () => void
  loading: boolean
}
```

### API Service Layer

Centralized API service for backend communication:

```typescript
class ApiService {
  private baseURL: string

  constructor() {
    this.baseURL = import.meta.env.VITE_API_BASE_URL
  }

  async get<T>(endpoint: string): Promise<T> {
    // Implementation
  }

  async post<T>(endpoint: string, data: any): Promise<T> {
    // Implementation
  }
}
```

### Data Fetching with TanStack Query

Efficient data fetching and caching:

```typescript
const { data: gpus, isLoading, error } = useQuery({
  queryKey: ['gpus', filters],
  queryFn: () => apiService.getGPUs(filters),
  staleTime: 5 * 60 * 1000, // 5 minutes
  refetchInterval: 30 * 1000, // 30 seconds
})
```

## Styling and Design

### Design System

The application follows a consistent design system based on:

- **Color Palette**: Cream-based monochromatic scheme
- **Typography**: Aeonik font family for modern aesthetics
- **Spacing**: 8px grid system for consistent layouts
- **Components**: Reusable UI components with variant support

### Component Architecture

```typescript
// Example component structure
interface ButtonProps {
  variant?: 'default' | 'secondary' | 'outline'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  children: React.ReactNode
}

const Button: React.FC<ButtonProps> = ({
  variant = 'default',
  size = 'md',
  ...props
}) => {
  return (
    <button
      className={cn(buttonVariants({ variant, size }))}
      {...props}
    />
  )
}
```

### Responsive Design

Mobile-first responsive design approach:

```css
/* Tailwind responsive breakpoints */
sm: 640px   /* Small devices */
md: 768px   /* Medium devices */
lg: 1024px  /* Large devices */
xl: 1280px  /* Extra large devices */
2xl: 1536px /* 2X large devices */
```

## Testing

### Unit Testing Setup

```bash
# Install testing dependencies
npm install --save-dev @testing-library/react @testing-library/jest-dom vitest
```

### Test Configuration

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
})
```

### Example Test

```typescript
import { render, screen } from '@testing-library/react'
import { Button } from '@/components/ui/button'

describe('Button Component', () => {
  it('renders correctly', () => {
    render(<Button>Click me</Button>)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })
})
```

### Running Tests

```bash
# Run all tests
npm run test

# Run tests in watch mode
npm run test:watch

# Generate coverage report
npm run test:coverage
```

## Performance Optimization

### Code Splitting

Automatic code splitting with dynamic imports:

```typescript
const Dashboard = lazy(() => import('./pages/Dashboard'))
const GPUMarketplace = lazy(() => import('./pages/GPUMarketplace'))
```

### Bundle Analysis

```bash
# Analyze bundle size
npm run build:analyze
```

### Performance Metrics

- **First Contentful Paint**: < 1.5s
- **Largest Contentful Paint**: < 2.5s
- **Cumulative Layout Shift**: < 0.1
- **First Input Delay**: < 100ms

## Security Considerations

### Environment Variables

Never commit sensitive data to version control:

```bash
# .env.example
VITE_API_BASE_URL=https://api.example.com
VITE_APP_NAME=DanteGPU
# Add other non-sensitive variables
```

### Content Security Policy

Implement CSP headers for enhanced security:

```html
<meta http-equiv="Content-Security-Policy"
      content="default-src 'self'; script-src 'self' 'unsafe-inline';">
```

### Authentication Security

- JWT token storage in httpOnly cookies
- Automatic token refresh mechanism
- Secure logout with token invalidation

## Deployment

### Production Checklist

- [ ] Environment variables configured
- [ ] Build optimization enabled
- [ ] Security headers implemented
- [ ] Performance monitoring setup
- [ ] Error tracking configured
- [ ] Analytics integration complete

### CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy Frontend
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run build
      - run: npm run test
```

## Contributing

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**: Follow coding standards and conventions
4. **Add tests**: Ensure new features are properly tested
5. **Commit changes**: `git commit -m 'Add amazing feature'`
6. **Push to branch**: `git push origin feature/amazing-feature`
7. **Open a Pull Request**: Provide detailed description of changes

### Code Standards

- **TypeScript**: Strict type checking enabled
- **ESLint**: Follow configured linting rules
- **Prettier**: Consistent code formatting
- **Conventional Commits**: Use semantic commit messages

### Pull Request Guidelines

- Provide clear description of changes
- Include relevant tests
- Update documentation if necessary
- Ensure all CI checks pass
- Request review from maintainers

## Troubleshooting

### Common Issues

#### Development Server Won't Start

```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

#### Build Failures

```bash
# Check TypeScript errors
npm run type-check

# Check for linting issues
npm run lint
```

#### Performance Issues

```bash
# Analyze bundle size
npm run build:analyze

# Check for memory leaks
npm run dev -- --inspect
```

### Getting Help

- **Documentation**: Check this README and inline code comments
- **Issues**: Search existing GitHub issues before creating new ones
- **Discussions**: Use GitHub Discussions for questions and ideas
- **Support**: Contact the development team for urgent issues

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- **React Team**: For the excellent React framework
- **Tailwind CSS**: For the utility-first CSS framework
- **Vite Team**: For the fast build tool
- **Open Source Community**: For the amazing ecosystem of tools and libraries

---

**DanteGPU Frontend** - Professional GPU rental platform interface built with modern web technologies.
