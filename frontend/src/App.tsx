import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { Suspense, lazy } from "react";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./components/auth/ProtectedRoute";
import { UserManagement } from "./components/auth/UserManagement";
import { Layout } from "./components/layout/Layout";
import { LoadingSpinner } from "./components/ui/LoadingSpinner";
import { ToastContainer } from "./components/ui/ToastContainer";
import { AuthProvider } from "./contexts/AuthContext";
import { ModalProvider } from "./contexts/ModalContext";
import { ToastProvider } from "./contexts/ToastContext";
import { queryClient } from "./lib/queryClient";

// Lazy load page components for code splitting
const Dashboard = lazy(() =>
	import("./pages/Dashboard").then((module) => ({ default: module.Dashboard })),
);
const QueuePage = lazy(() =>
	import("./pages/QueuePage").then((module) => ({ default: module.QueuePage })),
);
const HealthPage = lazy(() =>
	import("./pages/HealthPage").then((module) => ({ default: module.HealthPage })),
);
const FilesPage = lazy(() =>
	import("./pages/FilesPage").then((module) => ({ default: module.FilesPage })),
);
const ConfigurationPage = lazy(() =>
	import("./pages/ConfigurationPage").then((module) => ({ default: module.ConfigurationPage })),
);

// Loading fallback component
function PageLoader() {
	return (
		<div className="flex min-h-[400px] items-center justify-center">
			<LoadingSpinner size="lg" />
		</div>
	);
}

function App() {
	return (
		<QueryClientProvider client={queryClient}>
			<ToastProvider>
				<ModalProvider>
					<AuthProvider>
						<BrowserRouter>
							<div className="min-h-screen bg-base-100" data-theme="light">
								<Routes>
									{/* Protected routes */}
									<Route
										path="/"
										element={
											<ProtectedRoute>
												<Layout />
											</ProtectedRoute>
										}
									>
										<Route
											index
											element={
												<Suspense fallback={<PageLoader />}>
													<Dashboard />
												</Suspense>
											}
										/>
										<Route
											path="queue"
											element={
												<Suspense fallback={<PageLoader />}>
													<QueuePage />
												</Suspense>
											}
										/>
										<Route
											path="health"
											element={
												<Suspense fallback={<PageLoader />}>
													<HealthPage />
												</Suspense>
											}
										/>
										<Route
											path="files"
											element={
												<Suspense fallback={<PageLoader />}>
													<FilesPage />
												</Suspense>
											}
										/>

										{/* Admin-only routes */}
										<Route
											path="admin"
											element={
												<ProtectedRoute requireAdmin>
													<UserManagement />
												</ProtectedRoute>
											}
										/>
										<Route
											path="config"
											element={
												<ProtectedRoute requireAdmin>
													<Suspense fallback={<PageLoader />}>
														<ConfigurationPage />
													</Suspense>
												</ProtectedRoute>
											}
										/>
										<Route
											path="config/:section"
											element={
												<ProtectedRoute requireAdmin>
													<Suspense fallback={<PageLoader />}>
														<ConfigurationPage />
													</Suspense>
												</ProtectedRoute>
											}
										/>
									</Route>
								</Routes>
							</div>
							<ToastContainer />
						</BrowserRouter>
					</AuthProvider>
				</ModalProvider>
			</ToastProvider>
			<ReactQueryDevtools initialIsOpen={false} />
		</QueryClientProvider>
	);
}

export default App;
