import { QueryClient } from "react-query"
import { API } from "api/api"

export const createChat = (queryClient: QueryClient) => {
	return {
		mutationFn: API.createChat,
		onSuccess: async () => {
			await queryClient.invalidateQueries(["chats"]);
		},
	};
};

export const getChats = () => {
	return {
		queryKey: ["chats"],
		queryFn: API.getChats,
	};
};

export const getChatMessages = (chatID: string) => {
	return {
		queryKey: ["chatMessages", chatID],
		queryFn: () => API.getChatMessages(chatID),
	};
};
