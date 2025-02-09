import { createClient, createConfig, type ClientOptions } from "@hey-api/client-fetch";

export const client = createClient(createConfig<ClientOptions>({
    baseUrl: 'http://localhost:8080/api'
}));
