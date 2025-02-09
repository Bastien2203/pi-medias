"use server";

import {config} from "dotenv";
import {Media} from "@/types";


function url() {
    config();
    const API_HOST = process.env.API_HOST;
    const API_PORT = process.env.API_PORT;
    const API_PROTOCOL = process.env.API_PROTOCOL;
    return `${API_PROTOCOL}://${API_HOST}:${API_PORT}`;
}

export async function login(
    username: string,
    password: string
): Promise<string> {
    const response = await fetch(`${url()}/login`, {
        method: "POST",
        body: JSON.stringify({
            username: username,
            password: password
        }),
        headers: {
            "Content-Type": "application/json"
        }
    })


    if (!response.ok) {
        throw new Error("Invalid credentials");
    }
    const data = await response.json();
    if (data.error) {
        throw new Error(data.error);
    }
    return data.token;
}


export async function getAllMedias(token: string): Promise<Array<Media>> {
    const response = await fetch(`${url()}/media`, {
        headers: {
            "Authorization": `Bearer ${token}`
        }
    });

    if (!response.ok) {
        throw new Error("Unauthorized");
    }

    return (await response.json()) as Array<Media>;
}

export async function getMedia(mediaId: string, token: string): Promise<Media> {
    const response = await fetch(`${url()}/media/${mediaId}`, {
        headers: {
            "Authorization": `Bearer ${token}`
        },
    });

    if (!response.ok) {
        throw new Error("Unauthorized");
    }

    return (await response.json()) as Media;
}


export async function uploadMedia(file : File, name: string, token: string): Promise<Media> {
    const formData = new FormData();
    formData.append("file", file);
    const response = await fetch(`${url()}/media`, {
        method: "POST",
        headers: {
            "Authorization": `Bearer ${token}`,
            "Filename": name
        },
        body: formData
    });

    if (!response.ok) {
        throw new Error("Upload failed");
    }

    return (await response.json()) as Media;
}