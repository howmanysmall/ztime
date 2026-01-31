import { extname } from "node:path";
import type { Hooks, PluginInput } from "@opencode-ai/plugin";

type ShellExpression = PluginInput["$"] extends (first: TemplateStringsArray, ...rest: infer Rest) => unknown
	? Rest[number]
	: never;

const enum LogLevel {
	Debug = "debug",
	Error = "error",
	Info = "info",
	Warn = "warn",
}

// oxlint-disable-next-line id-length
export async function QualityCheckPlugin({ $, client }: PluginInput): Promise<Hooks> {
	async function logAsync(
		message: string,
		level: LogLevel = LogLevel.Info,
		extra?: Record<string, unknown>,
	): Promise<void> {
		try {
			await client.app.log({
				body: {
					level,
					message,
					service: "quality-check-plugin",
					...(extra ? { extra } : undefined),
				},
			});
		} catch {
			// nobody gaf
		}
	}
	function getCheckAsync(
		name: string,
	): (strings: TemplateStringsArray, ...expressions: ReadonlyArray<ShellExpression>) => Promise<void> {
		return async function checkAsync(
			strings: TemplateStringsArray,
			...expressions: ReadonlyArray<ShellExpression>
		): Promise<void> {
			await logAsync(`Running quality check: ${name}`);

			const shellOutput = await $(strings, ...expressions)
				.quiet()
				.nothrow();
			const { exitCode } = shellOutput;
			if (exitCode === 0) {
				await logAsync(`Quality check passed: ${name}`);
				return;
			}

			const stderr = await new Response(shellOutput.stderr).text();
			await logAsync(`Quality check failed: ${name}`, LogLevel.Error, {
				exitCode: exitCode,
				stderr,
				stdout: shellOutput.text(),
			});
			throw new Error(`Quality check "${name}" failed with exit code ${exitCode}${stderr ? `:\n${stderr}` : ""}`);
		};
	}

	const formatAsync = getCheckAsync("Format");
	const lintAsync = getCheckAsync("Lint");
	const testAsync = getCheckAsync("Test");
	const typeCheckAsync = getCheckAsync("Type Check");

	return {
		event: async ({ event }) => {
			if (event.type === "file.edited") {
				const { file } = event.properties;
				await logAsync(`Running quality checks on edited file: ${file}`);

				switch (extname(file).toLowerCase()) {
					case ".go":
						await formatAsync`go fmt ${file}`;
						await lintAsync`golangci-lint run ./...`;
						await typeCheckAsync`go vet ./...`;
						await testAsync`go test -v -race ./...`;
						break;

					default:
						break;
				}
			}
		},
	};
}
